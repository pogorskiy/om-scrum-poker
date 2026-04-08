package domain

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
)

// TimerState represents the current state of the room timer.
type TimerState string

const (
	TimerIdle    TimerState = "idle"
	TimerRunning TimerState = "running"
	TimerExpired TimerState = "expired"
)

const (
	MinTimerDuration     = 30
	MaxTimerDuration     = 600
	DefaultTimerDuration = 30
)

// Timer holds the countdown timer state for a room.
type Timer struct {
	Duration  int        // seconds, 30-600
	State     TimerState
	StartedAt time.Time // zero when idle/expired
}

// TimerSnapshot is a point-in-time view of the timer state.
type TimerSnapshot struct {
	Duration  int
	State     TimerState
	StartedAt time.Time
	Remaining int // seconds
}

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
	Role      string // "voter" or "observer"
	LastPing  time.Time
}

// Room holds the state for a single poker session.
type Room struct {
	ID           string
	Name         string
	CreatedBy    string // display name of the room creator
	Phase        Phase
	Participants map[string]*Participant // keyed by SessionID
	Timer        Timer
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

// NewRoom creates a room with the given ID, name, and creator session ID.
func NewRoom(id, name, createdBy string) (*Room, error) {
	if id == "" {
		return nil, fmt.Errorf("room id cannot be empty")
	}
	if len(name) > MaxRoomName {
		name = name[:MaxRoomName]
	}
	r := &Room{
		ID:           id,
		Name:         name,
		CreatedBy:    createdBy,
		Phase:        PhaseVoting,
		Participants: make(map[string]*Participant),
		Timer:        Timer{Duration: DefaultTimerDuration, State: TimerIdle},
		CreatedAt:    time.Now(),
	}
	r.TouchActivity()
	return r, nil
}

// Lock acquires the room mutex.
func (r *Room) Lock() { r.mu.Lock() }

// Unlock releases the room mutex.
func (r *Room) Unlock() { r.mu.Unlock() }

// maxCombiningMarks limits consecutive combining marks per base character
// to prevent zalgo text while allowing normal diacritics.
const maxCombiningMarks = 3

// sanitizeName removes control characters, invisible Unicode, zero-width
// characters, and excessive combining marks from a display name.
// Normal spaces (U+0020) are preserved.
func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	var b strings.Builder
	combiningCount := 0
	for _, r := range name {
		// Limit consecutive combining marks (prevents zalgo text).
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
			combiningCount++
			if combiningCount > maxCombiningMarks {
				continue
			}
			b.WriteRune(r)
			continue
		}
		combiningCount = 0

		// Keep normal printable characters (including space U+0020).
		if r == ' ' {
			b.WriteRune(r)
			continue
		}
		// Remove ASCII control characters (0x00-0x1F, 0x7F).
		if r <= 0x1F || r == 0x7F {
			continue
		}
		// Remove Unicode control characters (Cc, Cf).
		if unicode.Is(unicode.Cc, r) || unicode.Is(unicode.Cf, r) {
			continue
		}
		// Remove zero-width and invisible formatting characters.
		if r >= 0x200B && r <= 0x200F {
			continue
		}
		if r >= 0x2028 && r <= 0x202F {
			continue
		}
		if r >= 0x2060 && r <= 0x2069 {
			continue
		}
		if r == 0xFEFF {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// Join adds or re-joins a participant. Returns (participant, isNew).
// Role must be "voter" or "observer"; empty defaults to "voter".
// Must be called with lock held.
func (r *Room) Join(sessionID, name, role string) (*Participant, bool, error) {
	name = sanitizeName(name)
	if name == "" {
		return nil, false, fmt.Errorf("invalid_name: name cannot be empty")
	}
	if runes := []rune(name); len(runes) > MaxParticipantName {
		name = string(runes[:MaxParticipantName])
	}
	if role == "" {
		role = "voter"
	}
	if role != "voter" && role != "observer" {
		return nil, false, fmt.Errorf("invalid_role: %q is not a valid role", role)
	}

	if p, ok := r.Participants[sessionID]; ok {
		// Rejoin — restore and update name/role if changed.
		p.Name = name
		p.Role = role
		p.Status = "active"
		p.LastPing = time.Now()
		// Clear vote when switching to observer on rejoin.
		if role == "observer" {
			p.Vote = ""
		}
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
		Role:      role,
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

// UpdateRole changes a participant's role.
// Must be called with lock held.
func (r *Room) UpdateRole(sessionID, role string) error {
	if role != "voter" && role != "observer" {
		return fmt.Errorf("invalid_role: %q is not a valid role", role)
	}
	p, ok := r.Participants[sessionID]
	if !ok {
		return fmt.Errorf("room_not_found: participant not in room")
	}
	p.Role = role
	// Clear vote when switching to observer.
	if role == "observer" {
		p.Vote = ""
	}
	r.TouchActivity()
	return nil
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
	if p.Role == "observer" {
		return fmt.Errorf("invalid_vote: observers cannot vote")
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
// Returns error if already in voting phase (idempotency guard).
// Must be called with lock held.
func (r *Room) NewRound() error {
	if r.Phase == PhaseVoting {
		return fmt.Errorf("already in voting phase")
	}
	for _, p := range r.Participants {
		p.Vote = ""
	}
	r.Phase = PhaseVoting
	r.ResetTimer()
	r.TouchActivity()
	return nil
}

// ClearRoom removes all participants and resets the phase.
// Connected clients are expected to re-join automatically.
// Must be called with lock held.
func (r *Room) ClearRoom() {
	r.Participants = make(map[string]*Participant)
	r.Phase = PhaseVoting
	r.Timer = Timer{Duration: DefaultTimerDuration, State: TimerIdle}
	r.TouchActivity()
}

// UpdateName changes a participant's display name.
// Must be called with lock held.
func (r *Room) UpdateName(sessionID, name string) error {
	name = sanitizeName(name)
	if name == "" {
		return fmt.Errorf("invalid_name: name cannot be empty")
	}
	if runes := []rune(name); len(runes) > MaxParticipantName {
		name = string(runes[:MaxParticipantName])
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

// SetTimerDuration sets the timer duration in seconds. Only allowed when timer is idle.
// Duration must be in range [MinTimerDuration, MaxTimerDuration].
// Must be called with lock held.
func (r *Room) SetTimerDuration(d int) error {
	if r.Timer.State != TimerIdle {
		return fmt.Errorf("timer must be idle to change duration")
	}
	if d < MinTimerDuration || d > MaxTimerDuration {
		return fmt.Errorf("duration must be between %d and %d seconds", MinTimerDuration, MaxTimerDuration)
	}
	r.Timer.Duration = d
	r.TouchActivity()
	return nil
}

// StartTimer begins the countdown. Only allowed in idle state.
// Idempotent: no-op if already running. Returns true if state changed.
// Must be called with lock held.
func (r *Room) StartTimer() bool {
	if r.Timer.State != TimerIdle {
		return false
	}
	r.Timer.State = TimerRunning
	r.Timer.StartedAt = time.Now()
	r.TouchActivity()
	return true
}

// ResetTimer returns the timer to idle with the current duration.
// Idempotent: no-op if already idle. Returns true if state changed.
// Must be called with lock held.
func (r *Room) ResetTimer() bool {
	if r.Timer.State == TimerIdle {
		return false
	}
	r.Timer.State = TimerIdle
	r.Timer.StartedAt = time.Time{}
	r.TouchActivity()
	return true
}

// TimerInfo returns a snapshot of the timer state.
// Auto-transitions running -> expired if remaining time <= 0.
// Must be called with lock held.
func (r *Room) TimerInfo() TimerSnapshot {
	snap := TimerSnapshot{
		Duration:  r.Timer.Duration,
		State:     r.Timer.State,
		StartedAt: r.Timer.StartedAt,
	}

	switch r.Timer.State {
	case TimerIdle:
		snap.Remaining = r.Timer.Duration
	case TimerRunning:
		elapsed := int(time.Since(r.Timer.StartedAt).Seconds())
		remaining := r.Timer.Duration - elapsed
		if remaining <= 0 {
			// Auto-transition to expired.
			r.Timer.State = TimerExpired
			snap.State = TimerExpired
			snap.Remaining = 0
		} else {
			snap.Remaining = remaining
		}
	case TimerExpired:
		snap.Remaining = 0
	}

	return snap
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
