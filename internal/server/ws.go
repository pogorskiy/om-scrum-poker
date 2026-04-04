package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"nhooyr.io/websocket"

	"om-scrum-poker/internal/domain"
)

const maxMessageSize = 1024 // 1 KB

// HandleWebSocket upgrades HTTP to WebSocket and manages the connection lifecycle.
func HandleWebSocket(manager *RoomManager, limiter *RateLimiter, trustProxy bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r, trustProxy)
		if !limiter.AllowWSConnection(ip) {
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}

		// Extract room ID from path: /ws/{roomID}
		roomID := strings.TrimPrefix(r.URL.Path, "/ws/")
		if roomID == "" || roomID == r.URL.Path {
			http.Error(w, "missing room id", http.StatusBadRequest)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true, // Allow any origin for development.
		})
		if err != nil {
			log.Printf("websocket accept: %v", err)
			return
		}
		conn.SetReadLimit(maxMessageSize)

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		client := NewClient(conn, roomID, manager)

		// Start write pump in background.
		go func() {
			client.WritePump(ctx)
			cancel()
		}()

		// Read pump (blocking).
		readPump(ctx, client, manager, limiter, ip)

		// Cleanup on disconnect.
		client.Close()
		manager.UnregisterClient(roomID, client)

		if client.sessionID != "" {
			room := manager.GetRoom(roomID)
			if room != nil {
				room.Lock()
				if p, ok := room.Participants[client.sessionID]; ok {
					p.Status = "disconnected"
				}
				room.Unlock()

				msg, _ := MakeEnvelope("presence_changed", PresenceChangedPayload{
					SessionID: client.sessionID,
					Status:    "disconnected",
				})
				if msg != nil {
					manager.Broadcast(roomID, msg)
				}
			}
		}

		conn.Close(websocket.StatusNormalClosure, "goodbye")
	}
}

func readPump(ctx context.Context, client *Client, manager *RoomManager, limiter *RateLimiter, ip string) {
	for {
		_, data, err := client.conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				return
			}
			if ctx.Err() != nil {
				return
			}
			log.Printf("client %s: read error: %v", client.sessionID, err)
			return
		}

		env, err := ParseEnvelope(data)
		if err != nil {
			client.SendError("invalid_message", err.Error())
			continue
		}

		dispatch(ctx, client, manager, limiter, ip, env)
	}
}

func dispatch(ctx context.Context, client *Client, manager *RoomManager, limiter *RateLimiter, ip string, env *Envelope) {
	switch env.Type {
	case "join":
		handleJoin(client, manager, limiter, ip, env.Payload)
	case "vote":
		handleVote(client, manager, env.Payload)
	case "reveal":
		handleReveal(client, manager)
	case "new_round":
		handleNewRound(client, manager)
	case "clear_room":
		handleClearRoom(client, manager)
	case "update_name":
		handleUpdateName(client, manager, env.Payload)
	case "presence":
		handlePresence(client, manager, env.Payload)
	case "leave":
		handleLeave(client, manager)
	default:
		client.SendError("invalid_message", "unknown event type: "+env.Type)
	}
}

func handleJoin(client *Client, manager *RoomManager, limiter *RateLimiter, ip string, payload json.RawMessage) {
	var p JoinPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		client.SendError("invalid_message", "invalid join payload")
		return
	}
	if p.SessionID == "" {
		client.SendError("invalid_message", "sessionId is required")
		return
	}
	if p.UserName == "" {
		client.SendError("invalid_name", "userName is required")
		return
	}

	// Check room creation rate limit.
	existing := manager.GetRoom(client.roomID)
	if existing == nil {
		if !limiter.AllowRoomCreation(ip) {
			client.SendError("rate_limited", "too many room creations")
			return
		}
	}

	room, err := manager.GetOrCreateRoom(client.roomID, p.UserName+"'s Room")
	if err != nil {
		client.SendError("room_not_found", err.Error())
		return
	}

	room.Lock()
	participant, isNew, err := room.Join(p.SessionID, p.UserName)
	if err != nil {
		room.Unlock()
		errCode := "invalid_name"
		if strings.Contains(err.Error(), "room_full") {
			errCode = "room_full"
		}
		client.SendError(errCode, err.Error())
		return
	}
	_ = participant

	client.sessionID = p.SessionID
	manager.RegisterClient(client.roomID, client)

	// Send full room state to the joining client.
	state := manager.BuildRoomState(room)
	room.Unlock()

	stateMsg, _ := MakeEnvelope("room_state", state)
	if stateMsg != nil {
		client.Send(stateMsg)
	}

	// Broadcast join to others.
	if isNew {
		joinMsg, _ := MakeEnvelope("participant_joined", ParticipantJoinedPayload{
			SessionID: p.SessionID,
			UserName:  p.UserName,
			Status:    "active",
		})
		if joinMsg != nil {
			manager.BroadcastExcept(client.roomID, joinMsg, client)
		}
	} else {
		// Rejoin — broadcast presence change.
		presMsg, _ := MakeEnvelope("presence_changed", PresenceChangedPayload{
			SessionID: p.SessionID,
			Status:    "active",
		})
		if presMsg != nil {
			manager.BroadcastExcept(client.roomID, presMsg, client)
		}
	}
}

func handleVote(client *Client, manager *RoomManager, payload json.RawMessage) {
	if client.sessionID == "" {
		client.SendError("invalid_message", "must join before voting")
		return
	}

	var p VotePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		client.SendError("invalid_vote", "invalid vote payload")
		return
	}

	room := manager.GetRoom(client.roomID)
	if room == nil {
		client.SendError("room_not_found", "room does not exist")
		return
	}

	room.Lock()
	hadVote := room.HasVoted(client.sessionID)
	err := room.CastVote(client.sessionID, domain.VoteValue(p.Value))
	room.Unlock()

	if err != nil {
		client.SendError("invalid_vote", err.Error())
		return
	}

	if p.Value == "" && hadVote {
		msg, _ := MakeEnvelope("vote_retracted", VoteRetractedPayload{SessionID: client.sessionID})
		if msg != nil {
			manager.Broadcast(client.roomID, msg)
		}
	} else if p.Value != "" {
		msg, _ := MakeEnvelope("vote_cast", VoteCastPayload{SessionID: client.sessionID})
		if msg != nil {
			manager.Broadcast(client.roomID, msg)
		}
	}
}

func handleReveal(client *Client, manager *RoomManager) {
	if client.sessionID == "" {
		client.SendError("invalid_message", "must join first")
		return
	}

	room := manager.GetRoom(client.roomID)
	if room == nil {
		client.SendError("room_not_found", "room does not exist")
		return
	}

	room.Lock()
	result, err := room.Reveal()
	room.Unlock()

	if err != nil {
		client.SendError("invalid_message", err.Error())
		return
	}

	msg, _ := MakeEnvelope("votes_revealed", result)
	if msg != nil {
		manager.Broadcast(client.roomID, msg)
	}
}

func handleNewRound(client *Client, manager *RoomManager) {
	if client.sessionID == "" {
		client.SendError("invalid_message", "must join first")
		return
	}

	room := manager.GetRoom(client.roomID)
	if room == nil {
		client.SendError("room_not_found", "room does not exist")
		return
	}

	room.Lock()
	room.NewRound()
	room.Unlock()

	msg, _ := MakeEnvelope("round_reset", struct{}{})
	if msg != nil {
		manager.Broadcast(client.roomID, msg)
	}
}

func handleClearRoom(client *Client, manager *RoomManager) {
	if client.sessionID == "" {
		client.SendError("invalid_message", "must join first")
		return
	}

	room := manager.GetRoom(client.roomID)
	if room == nil {
		client.SendError("room_not_found", "room does not exist")
		return
	}

	room.Lock()
	room.ClearRoom()
	room.Unlock()

	msg, _ := MakeEnvelope("room_cleared", struct{}{})
	if msg != nil {
		manager.Broadcast(client.roomID, msg)
	}
}

func handleUpdateName(client *Client, manager *RoomManager, payload json.RawMessage) {
	if client.sessionID == "" {
		client.SendError("invalid_message", "must join first")
		return
	}

	var p UpdateNamePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		client.SendError("invalid_name", "invalid payload")
		return
	}

	room := manager.GetRoom(client.roomID)
	if room == nil {
		client.SendError("room_not_found", "room does not exist")
		return
	}

	room.Lock()
	err := room.UpdateName(client.sessionID, p.UserName)
	room.Unlock()

	if err != nil {
		client.SendError("invalid_name", err.Error())
		return
	}

	msg, _ := MakeEnvelope("name_updated", NameUpdatedPayload{
		SessionID: client.sessionID,
		UserName:  p.UserName,
	})
	if msg != nil {
		manager.Broadcast(client.roomID, msg)
	}
}

func handlePresence(client *Client, manager *RoomManager, payload json.RawMessage) {
	if client.sessionID == "" {
		client.SendError("invalid_message", "must join first")
		return
	}

	var p PresencePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		client.SendError("invalid_message", "invalid payload")
		return
	}

	room := manager.GetRoom(client.roomID)
	if room == nil {
		client.SendError("room_not_found", "room does not exist")
		return
	}

	room.Lock()
	err := room.UpdatePresence(client.sessionID, p.Status)
	room.Unlock()

	if err != nil {
		client.SendError("invalid_message", err.Error())
		return
	}

	msg, _ := MakeEnvelope("presence_changed", PresenceChangedPayload{
		SessionID: client.sessionID,
		Status:    p.Status,
	})
	if msg != nil {
		manager.Broadcast(client.roomID, msg)
	}
}

func handleLeave(client *Client, manager *RoomManager) {
	if client.sessionID == "" {
		return
	}

	room := manager.GetRoom(client.roomID)
	if room == nil {
		return
	}

	room.Lock()
	room.Leave(client.sessionID)
	room.Unlock()

	msg, _ := MakeEnvelope("participant_left", ParticipantLeftPayload{SessionID: client.sessionID})
	if msg != nil {
		manager.BroadcastExcept(client.roomID, msg, client)
	}

	manager.UnregisterClient(client.roomID, client)
	client.sessionID = ""
}

// clientIP extracts the client IP from the request.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[0])
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return xri
		}
	}
	// Strip port from RemoteAddr.
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}
