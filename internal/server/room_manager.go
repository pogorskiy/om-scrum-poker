package server

import (
	"log"
	"sort"
	"sync"
	"time"

	"om-scrum-poker/internal/domain"
)

const (
	gcInterval     = 10 * time.Minute
	roomExpiry     = 24 * time.Hour
)

// RoomManager holds all active rooms and their connected clients.
type RoomManager struct {
	mu      sync.RWMutex
	rooms   map[string]*domain.Room
	clients map[string]map[*Client]struct{} // roomID → set of clients
	startAt time.Time
}

// NewRoomManager creates a new manager.
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms:   make(map[string]*domain.Room),
		clients: make(map[string]map[*Client]struct{}),
		startAt: time.Now(),
	}
}

// StartGC launches a background goroutine that removes stale rooms.
// It returns a stop function.
func (rm *RoomManager) StartGC() func() {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(gcInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				rm.collectGarbage()
			}
		}
	}()
	return func() { close(stop) }
}

func (rm *RoomManager) collectGarbage() {
	// Phase 1: collect candidates under read lock
	rm.mu.RLock()
	now := time.Now()
	var candidates []string
	for id, room := range rm.rooms {
		clients := rm.clients[id]
		if len(clients) == 0 && now.Sub(room.GetLastActivity()) > roomExpiry {
			candidates = append(candidates, id)
		}
	}
	rm.mu.RUnlock()

	if len(candidates) == 0 {
		return
	}

	// Phase 2: delete each candidate under write lock with re-check
	rm.mu.Lock()
	defer rm.mu.Unlock()
	for _, id := range candidates {
		room, ok := rm.rooms[id]
		if !ok {
			continue
		}
		clients := rm.clients[id]
		if len(clients) == 0 && now.Sub(room.GetLastActivity()) > roomExpiry {
			delete(rm.rooms, id)
			delete(rm.clients, id)
			log.Printf("GC: removed stale room %s", id)
		}
	}
}

// GetOrCreateRoom returns an existing room or creates a new one.
func (rm *RoomManager) GetOrCreateRoom(id, name, createdBy string) (*domain.Room, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if room, ok := rm.rooms[id]; ok {
		return room, nil
	}

	room, err := domain.NewRoom(id, name, createdBy)
	if err != nil {
		return nil, err
	}
	rm.rooms[id] = room
	rm.clients[id] = make(map[*Client]struct{})
	return room, nil
}

// GetRoom returns a room if it exists.
func (rm *RoomManager) GetRoom(id string) *domain.Room {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.rooms[id]
}

// RegisterClient adds a client to a room's client set.
func (rm *RoomManager) RegisterClient(roomID string, client *Client) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if _, ok := rm.clients[roomID]; !ok {
		rm.clients[roomID] = make(map[*Client]struct{})
	}
	rm.clients[roomID][client] = struct{}{}
}

// UnregisterClient removes a client from a room's client set.
func (rm *RoomManager) UnregisterClient(roomID string, client *Client) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if clients, ok := rm.clients[roomID]; ok {
		delete(clients, client)
	}
}

// Broadcast sends a message to all clients in a room.
func (rm *RoomManager) Broadcast(roomID string, msg []byte) {
	rm.mu.RLock()
	clients := rm.clients[roomID]
	// Copy to avoid holding lock during sends.
	list := make([]*Client, 0, len(clients))
	for c := range clients {
		list = append(list, c)
	}
	rm.mu.RUnlock()

	for _, c := range list {
		c.Send(msg)
	}
}

// BroadcastExcept sends a message to all clients except the given one.
func (rm *RoomManager) BroadcastExcept(roomID string, msg []byte, except *Client) {
	rm.mu.RLock()
	clients := rm.clients[roomID]
	list := make([]*Client, 0, len(clients))
	for c := range clients {
		if c != except {
			list = append(list, c)
		}
	}
	rm.mu.RUnlock()

	for _, c := range list {
		c.Send(msg)
	}
}

// UpdatePingTime updates the last ping time for a participant.
func (rm *RoomManager) UpdatePingTime(roomID, sessionID string) {
	room := rm.GetRoom(roomID)
	if room == nil {
		return
	}
	room.Lock()
	defer room.Unlock()
	if p, ok := room.Participants[sessionID]; ok {
		p.LastPing = time.Now()
	}
}

// RoomCount returns the number of active rooms.
func (rm *RoomManager) RoomCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return len(rm.rooms)
}

// ConnectionCount returns the total number of connected clients.
func (rm *RoomManager) ConnectionCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	total := 0
	for _, clients := range rm.clients {
		total += len(clients)
	}
	return total
}

// Uptime returns the duration since the manager was created.
func (rm *RoomManager) Uptime() time.Duration {
	return time.Since(rm.startAt)
}

// CloseAll disconnects all clients gracefully with StatusGoingAway.
// It sends close frames concurrently and waits up to 3 seconds.
func (rm *RoomManager) CloseAll() {
	rm.mu.Lock()
	var allClients []*Client
	for _, clients := range rm.clients {
		for c := range clients {
			allClients = append(allClients, c)
		}
	}
	rm.mu.Unlock()

	// Send close frames concurrently with a timeout.
	var wg sync.WaitGroup
	for _, c := range allClients {
		wg.Add(1)
		go func(client *Client) {
			defer wg.Done()
			client.CloseGraceful()
		}(c)
	}

	// Wait up to 3 seconds for all close frames to be sent.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		log.Println("CloseAll: timed out waiting for graceful close")
	}
}

// BuildRoomState creates a RoomStatePayload for sending to a client.
func (rm *RoomManager) BuildRoomState(room *domain.Room) RoomStatePayload {
	// Must be called with room lock held.
	participants := make([]ParticipantInfo, 0, len(room.Participants))
	for _, p := range room.Participants {
		participants = append(participants, ParticipantInfo{
			SessionID: p.SessionID,
			UserName:  p.Name,
			Status:    p.Status,
			HasVoted:  p.Vote != "",
			Role:      p.Role,
		})
	}
	// Sort by join time (stable order, not by random sessionId).
	sort.Slice(participants, func(i, j int) bool {
		ti := room.Participants[participants[i].SessionID].JoinedAt
		tj := room.Participants[participants[j].SessionID].JoinedAt
		if ti.Equal(tj) {
			return participants[i].SessionID < participants[j].SessionID
		}
		return ti.Before(tj)
	})

	state := RoomStatePayload{
		RoomID:       room.ID,
		RoomName:     room.Name,
		CreatedBy:    room.CreatedBy,
		Phase:        room.Phase,
		Participants: participants,
	}

	if room.Phase == domain.PhaseReveal {
		state.Result = domain.CalculateResult(room.Participants)
	}

	snapshot := room.TimerInfo()
	state.Timer = buildTimerPayload(snapshot)

	return state
}
