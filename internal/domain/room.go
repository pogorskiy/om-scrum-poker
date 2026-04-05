package domain

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Phase represents the current state of a poker room.
type Phase string

const (
	PhaseVoting Phase = "voting"
	PhaseReveal Phase = "reveal"
)

// VoteValue represents a valid card value or empty string (no vote).
type VoteValue string

// Valid card values for scrum poker.
var ValidVotes = map[VoteValue]bool{
	"":    true, // retract
	"?":   true,
	"0":   true,
	"0.5": true,
	"1":   true,
	"2":   true,
	"3":   true,
	"5":   true,
	"8":   true,
	"13":  true,
	"20":  true,
	"40":  true,
	"100": true,
}

const (
	MaxRoomName        = 60
	MaxParticipantName = 30
	MaxParticipants    = 50
)

// Participant represents a user in a poker room.
type Participant struct {
	SessionID string
	Name      string
	Vote      VoteValue
	Status    string // "active", "idle", "disconnected"
	LastPing  time.Time
}

// Room holds the state for a single poker session.
type Room struct {
	ID           string
	Name         string
	Phase        Phase
	Participants map[string]*Participant // keyed by SessionID
	CreatedAt    time.Time
	lastActivity atomic.Int64
	mu           sync.Mutex
}

// TouchActivity updates the last activity timestamp to now.
func (r *Room) TouchActivity() {
	r.lastActivity.Store(time.Now().UnixNano())
}

// GetLastActivity returns the last activity time.
func (r *Room) GetLastActivity() time.Time {
	return time.Unix(0, r.lastActivity.Load())
}

// LastActivityUnixNano returns the raw atomic value of last activity.
func (r *Room) LastActivityUnixNano() int64 {
	return r.lastActivity.Load()
}

// SetLastActivity sets the last activity timestamp to the given time.
func (r *Room) SetLastActivity(t time.Time) {
	r.lastActivity.Store(t.UnixNano())
}

// NewRoom creates a room with the given ID and name.
func NewRoom(id, name string) (*Room, error) {
	if id == "" {
		return nil, fmt.Errorf("room id cannot be empty")
	}
	if len(name) > MaxRoomName {
		name = name[:MaxRoomName]
	}
	r := &Room{
		ID:           id,
		Name:         name,
		Phase:        PhaseVoting,
		Participants: make(map[string]*Participant),
		CreatedAt:    time.Now(),
	}
	r.TouchActivity()
	return r, nil
}

// Lock acquires the room mutex.
func (r *Room) Lock() { r.mu.Lock() }

// Unlock releases the room mutex.
func (r *Room) Unlock() { r.mu.Unlock() }

// Join adds or re-joins a participant. Returns (participant, isNew).
// Must be called with lock held.
func (r *Room) Join(sessionID, name string) (*Participant, bool, error) {
	if name == "" {
		return nil, false, fmt.Errorf("invalid_name: name cannot be empty")
	}
	if len(name) > MaxParticipantName {
		name = name[:MaxParticipantName]
	}

	if p, ok := r.Participants[sessionID]; ok {
		// Rejoin — restore and update name if changed.
		p.Name = name
		p.Status = "active"
		p.LastPing = time.Now()
		r.TouchActivity()
		return p, false, nil
	}

	if len(r.Participants) >= MaxParticipants {
		return nil, false, fmt.Errorf("room_full: room has reached maximum capacity")
	}

	p := &Participant{
		SessionID: sessionID,
		Name:      name,
		Vote:      "",
		Status:    "active",
		LastPing:  time.Now(),
	}
	r.Participants[sessionID] = p
	r.TouchActivity()
	return p, true, nil
}

// Leave removes a participant. Returns true if participant existed.
// Must be called with lock held.
func (r *Room) Leave(sessionID string) bool {
	if _, ok := r.Participants[sessionID]; !ok {
		return false
	}
	delete(r.Participants, sessionID)
	r.TouchActivity()
	return true
}

// CastVote sets a participant's vote. Empty string retracts.
// Must be called with lock held.
func (r *Room) CastVote(sessionID string, value VoteValue) error {
	if r.Phase != PhaseVoting {
		return fmt.Errorf("invalid_vote: voting is locked during reveal phase")
	}
	if !ValidVotes[value] {
		return fmt.Errorf("invalid_vote: %q is not a valid card value", value)
	}
	p, ok := r.Participants[sessionID]
	if !ok {
		return fmt.Errorf("room_not_found: participant not in room")
	}
	p.Vote = value
	r.TouchActivity()
	return nil
}

// Reveal transitions the room to reveal phase and returns round result.
// Must be called with lock held.
func (r *Room) Reveal() (*RoundResult, error) {
	if r.Phase == PhaseReveal {
		return nil, fmt.Errorf("already in reveal phase")
	}
	r.Phase = PhaseReveal
	r.TouchActivity()
	return CalculateResult(r.Participants), nil
}

// NewRound clears all votes and returns to voting phase.
// Must be called with lock held.
func (r *Room) NewRound() {
	for _, p := range r.Participants {
		p.Vote = ""
	}
	r.Phase = PhaseVoting
	r.TouchActivity()
}

// ClearRoom removes all participants and resets the phase.
// Connected clients are expected to re-join automatically.
// Must be called with lock held.
func (r *Room) ClearRoom() {
	r.Participants = make(map[string]*Participant)
	r.Phase = PhaseVoting
	r.TouchActivity()
}

// UpdateName changes a participant's display name.
// Must be called with lock held.
func (r *Room) UpdateName(sessionID, name string) error {
	if name == "" {
		return fmt.Errorf("invalid_name: name cannot be empty")
	}
	if len(name) > MaxParticipantName {
		name = name[:MaxParticipantName]
	}
	p, ok := r.Participants[sessionID]
	if !ok {
		return fmt.Errorf("room_not_found: participant not in room")
	}
	p.Name = name
	r.TouchActivity()
	return nil
}

// UpdatePresence updates a participant's status.
// Must be called with lock held.
func (r *Room) UpdatePresence(sessionID, status string) error {
	if status != "active" && status != "idle" {
		return fmt.Errorf("invalid status: %q", status)
	}
	p, ok := r.Participants[sessionID]
	if !ok {
		return fmt.Errorf("room_not_found: participant not in room")
	}
	p.Status = status
	p.LastPing = time.Now()
	r.TouchActivity()
	return nil
}

// HasVoted returns true if the participant has a non-empty vote.
func (r *Room) HasVoted(sessionID string) bool {
	p, ok := r.Participants[sessionID]
	if !ok {
		return false
	}
	return p.Vote != ""
}

// ActiveConnections returns the number of non-disconnected participants.
// Must be called with lock held.
func (r *Room) ActiveConnections() int {
	count := 0
	for _, p := range r.Participants {
		if p.Status != "disconnected" {
			count++
		}
	}
	return count
}
