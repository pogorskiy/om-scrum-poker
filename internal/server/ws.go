package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"

	"nhooyr.io/websocket"

	"om-scrum-poker/internal/domain"
)

// Compiled regexps for input validation.
var (
	validRoomID    = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)
	validSessionID = regexp.MustCompile(`^[a-f0-9]{32}$`)
)

const maxMessageSize = 4096 // 4 KB

// buildAcceptOptions creates WebSocket accept options based on allowed origins.
// If origins contains "*", all origins are allowed (InsecureSkipVerify).
// If origins is non-empty, only the specified origin patterns are allowed.
// If origins is empty, the default same-origin check is used.
func buildAcceptOptions(allowedOrigins []string) *websocket.AcceptOptions {
	for _, o := range allowedOrigins {
		if o == "*" {
			return &websocket.AcceptOptions{InsecureSkipVerify: true}
		}
	}
	if len(allowedOrigins) > 0 {
		return &websocket.AcceptOptions{OriginPatterns: allowedOrigins}
	}
	return &websocket.AcceptOptions{}
}

// HandleWebSocket upgrades HTTP to WebSocket and manages the connection lifecycle.
func HandleWebSocket(manager *RoomManager, limiter *RateLimiter, trustProxy bool, allowedOrigins []string) http.HandlerFunc {
	acceptOpts := buildAcceptOptions(allowedOrigins)
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
		if !validRoomID.MatchString(roomID) {
			http.Error(w, "invalid room id", http.StatusBadRequest)
			return
		}

		conn, err := websocket.Accept(w, r, acceptOpts)
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

		sid := client.SessionID()
		if sid != "" {
			room := manager.GetRoom(roomID)
			if room != nil {
				room.Lock()
				if p, ok := room.Participants[sid]; ok {
					p.Status = "disconnected"
				}
				room.Unlock()

				msg, _ := MakeEnvelope("presence_changed", PresenceChangedPayload{
					SessionID: sid,
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
			log.Printf("client %s: read error: %v", client.SessionID(), err)
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
	case "update_role":
		handleUpdateRole(client, manager, env.Payload)
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
	if !validSessionID.MatchString(p.SessionID) {
		client.SendError("invalid_message", "invalid sessionId format")
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

	roomName := p.RoomName
	if roomName == "" {
		roomName = p.UserName + "'s Room"
	}
	room, err := manager.GetOrCreateRoom(client.roomID, roomName, p.UserName)
	if err != nil {
		client.SendError("room_not_found", err.Error())
		return
	}

	room.Lock()
	participant, isNew, err := room.Join(p.SessionID, p.UserName, p.Role)
	if err != nil {
		room.Unlock()
		errCode := "invalid_name"
		if strings.Contains(err.Error(), "room_full") {
			errCode = "room_full"
		}
		client.SendError(errCode, err.Error())
		return
	}
	client.SetSessionID(p.SessionID)
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
			Role:      participant.Role,
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
	if client.SessionID() == "" {
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
	hadVote := room.HasVoted(client.SessionID())
	err := room.CastVote(client.SessionID(), domain.VoteValue(p.Value))
	room.Unlock()

	if err != nil {
		client.SendError("invalid_vote", err.Error())
		return
	}

	if p.Value == "" && hadVote {
		msg, _ := MakeEnvelope("vote_retracted", VoteRetractedPayload{SessionID: client.SessionID()})
		if msg != nil {
			manager.Broadcast(client.roomID, msg)
		}
	} else if p.Value != "" {
		msg, _ := MakeEnvelope("vote_cast", VoteCastPayload{SessionID: client.SessionID()})
		if msg != nil {
			manager.Broadcast(client.roomID, msg)
		}
	}
}

func handleReveal(client *Client, manager *RoomManager) {
	if client.SessionID() == "" {
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
	if client.SessionID() == "" {
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
	if client.SessionID() == "" {
		client.SendError("invalid_message", "must join first")
		return
	}

	room := manager.GetRoom(client.roomID)
	if room == nil {
		client.SendError("room_not_found", "room does not exist")
		return
	}

	// Broadcast before clearing so all connected clients receive the event.
	msg, _ := MakeEnvelope("room_cleared", struct{}{})
	if msg != nil {
		manager.Broadcast(client.roomID, msg)
	}

	room.Lock()
	room.ClearRoom()
	room.Unlock()
}

func handleUpdateName(client *Client, manager *RoomManager, payload json.RawMessage) {
	if client.SessionID() == "" {
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
	err := room.UpdateName(client.SessionID(), p.UserName)
	room.Unlock()

	if err != nil {
		client.SendError("invalid_name", err.Error())
		return
	}

	msg, _ := MakeEnvelope("name_updated", NameUpdatedPayload{
		SessionID: client.SessionID(),
		UserName:  p.UserName,
	})
	if msg != nil {
		manager.Broadcast(client.roomID, msg)
	}
}

func handlePresence(client *Client, manager *RoomManager, payload json.RawMessage) {
	if client.SessionID() == "" {
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
	err := room.UpdatePresence(client.SessionID(), p.Status)
	room.Unlock()

	if err != nil {
		client.SendError("invalid_message", err.Error())
		return
	}

	msg, _ := MakeEnvelope("presence_changed", PresenceChangedPayload{
		SessionID: client.SessionID(),
		Status:    p.Status,
	})
	if msg != nil {
		manager.Broadcast(client.roomID, msg)
	}
}

func handleUpdateRole(client *Client, manager *RoomManager, payload json.RawMessage) {
	if client.SessionID() == "" {
		client.SendError("invalid_message", "must join first")
		return
	}

	var p UpdateRolePayload
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
	err := room.UpdateRole(client.SessionID(), p.Role)
	room.Unlock()

	if err != nil {
		client.SendError("invalid_message", err.Error())
		return
	}

	msg, _ := MakeEnvelope("role_updated", RoleUpdatedPayload{
		SessionID: client.SessionID(),
		Role:      p.Role,
	})
	if msg != nil {
		manager.Broadcast(client.roomID, msg)
	}
}

func handleLeave(client *Client, manager *RoomManager) {
	if client.SessionID() == "" {
		return
	}

	room := manager.GetRoom(client.roomID)
	if room == nil {
		return
	}

	room.Lock()
	room.Leave(client.SessionID())
	room.Unlock()

	msg, _ := MakeEnvelope("participant_left", ParticipantLeftPayload{SessionID: client.SessionID()})
	if msg != nil {
		manager.BroadcastExcept(client.roomID, msg, client)
	}

	manager.UnregisterClient(client.roomID, client)
	client.SetSessionID("")
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
