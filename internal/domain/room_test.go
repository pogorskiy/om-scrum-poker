package domain

import (
	"testing"
	"time"
)

func TestNewRoom(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		rName   string
		wantErr bool
		wantLen int // expected name length
	}{
		{"valid room", "abc123", "Sprint 42", false, 9},
		{"empty id", "", "Sprint", true, 0},
		{"long name truncated", "abc", string(make([]byte, 100)), false, MaxRoomName},
		{"empty name ok", "abc", "", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewRoom(tt.id, tt.rName, "")
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewRoom() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if r.Phase != PhaseVoting {
				t.Errorf("expected phase %q, got %q", PhaseVoting, r.Phase)
			}
			if len(r.Name) != tt.wantLen {
				t.Errorf("expected name length %d, got %d", tt.wantLen, len(r.Name))
			}
		})
	}
}

func TestJoin(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")

	tests := []struct {
		name      string
		sessionID string
		userName  string
		wantNew   bool
		wantErr   bool
	}{
		{"first join", "s1", "Alice", true, false},
		{"second join", "s2", "Bob", true, false},
		{"rejoin same session", "s1", "Alice Updated", false, false},
		{"empty name", "s3", "", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, isNew, err := r.Join(tt.sessionID, tt.userName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Join() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if isNew != tt.wantNew {
				t.Errorf("Join() isNew = %v, want %v", isNew, tt.wantNew)
			}
			if p.Name != tt.userName {
				// Name may be truncated, check prefix.
				if len(tt.userName) <= MaxParticipantName && p.Name != tt.userName {
					t.Errorf("Join() name = %q, want %q", p.Name, tt.userName)
				}
			}
		})
	}
}

func TestJoinRejoinRestoresStatus(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice")

	// Simulate disconnect.
	r.Participants["s1"].Status = "disconnected"

	p, isNew, err := r.Join("s1", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	if isNew {
		t.Error("expected rejoin, got new")
	}
	if p.Status != "active" {
		t.Errorf("expected status active after rejoin, got %q", p.Status)
	}
}

func TestJoinRoomFull(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	for i := 0; i < MaxParticipants; i++ {
		_, _, err := r.Join(string(rune('A'+i))+"id", "User")
		if err != nil {
			t.Fatalf("unexpected error filling room: %v", err)
		}
	}
	_, _, err := r.Join("overflow", "Overflow")
	if err == nil {
		t.Fatal("expected room_full error")
	}
}

func TestLeave(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice")

	if !r.Leave("s1") {
		t.Error("expected Leave to return true for existing participant")
	}
	if r.Leave("s1") {
		t.Error("expected Leave to return false for non-existing participant")
	}
	if len(r.Participants) != 0 {
		t.Errorf("expected 0 participants, got %d", len(r.Participants))
	}
}

func TestCastVote(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice")

	tests := []struct {
		name    string
		value   VoteValue
		wantErr bool
	}{
		{"valid vote 5", "5", false},
		{"valid vote ?", "?", false},
		{"valid retract", "", false},
		{"invalid vote", "999", true},
		{"valid vote 0.5", "0.5", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.CastVote("s1", tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("CastVote() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCastVoteDuringReveal(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice")
	r.CastVote("s1", "5")
	r.Reveal()

	err := r.CastVote("s1", "8")
	if err == nil {
		t.Error("expected error when voting during reveal phase")
	}
}

func TestCastVoteNonExistentParticipant(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	err := r.CastVote("ghost", "5")
	if err == nil {
		t.Error("expected error for non-existent participant")
	}
}

func TestReveal(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice")
	r.CastVote("s1", "5")

	result, err := r.Reveal()
	if err != nil {
		t.Fatal(err)
	}
	if r.Phase != PhaseReveal {
		t.Errorf("expected phase %q, got %q", PhaseReveal, r.Phase)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Double reveal should error.
	_, err = r.Reveal()
	if err == nil {
		t.Error("expected error on double reveal")
	}
}

func TestNewRound(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice")
	r.Join("s2", "Bob")
	r.CastVote("s1", "5")
	r.CastVote("s2", "8")
	r.Reveal()

	r.NewRound()

	if r.Phase != PhaseVoting {
		t.Errorf("expected phase %q, got %q", PhaseVoting, r.Phase)
	}
	for _, p := range r.Participants {
		if p.Vote != "" {
			t.Errorf("expected empty vote after new round, got %q for %s", p.Vote, p.Name)
		}
	}
}

func TestClearRoom(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice")
	r.Join("s2", "Bob")
	r.CastVote("s1", "5")
	r.CastVote("s2", "8")

	r.ClearRoom()

	if len(r.Participants) != 0 {
		t.Errorf("expected 0 participants after clear, got %d", len(r.Participants))
	}
	if r.Phase != PhaseVoting {
		t.Errorf("expected phase %q, got %q", PhaseVoting, r.Phase)
	}
}

func TestUpdateName(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice")

	tests := []struct {
		name      string
		sessionID string
		newName   string
		wantErr   bool
	}{
		{"valid update", "s1", "Alice B", false},
		{"empty name", "s1", "", true},
		{"non-existent", "s99", "Ghost", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.UpdateName(tt.sessionID, tt.newName)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateName_Success(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice")

	err := r.UpdateName("s1", "Bob")
	if err != nil {
		t.Fatalf("UpdateName() unexpected error: %v", err)
	}
	if r.Participants["s1"].Name != "Bob" {
		t.Errorf("expected name %q, got %q", "Bob", r.Participants["s1"].Name)
	}
}

func TestUpdateName_LongName(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice")

	longName := "ABCDEFGHIJKLMNOPQRSTUVWXYZ12345678" // 34 chars, exceeds MaxParticipantName (30)
	err := r.UpdateName("s1", longName)
	if err != nil {
		t.Fatalf("UpdateName() unexpected error: %v", err)
	}
	got := r.Participants["s1"].Name
	if len(got) != MaxParticipantName {
		t.Errorf("expected name length %d, got %d", MaxParticipantName, len(got))
	}
	if got != longName[:MaxParticipantName] {
		t.Errorf("expected truncated name %q, got %q", longName[:MaxParticipantName], got)
	}
}

func TestUpdateName_UpdatesLastActivity(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice")

	before := r.GetLastActivity()
	time.Sleep(1 * time.Millisecond)

	err := r.UpdateName("s1", "Bob")
	if err != nil {
		t.Fatalf("UpdateName() unexpected error: %v", err)
	}
	if !r.GetLastActivity().After(before) {
		t.Error("expected LastActivity to be updated after name change")
	}
}

func TestUpdatePresence(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice")

	tests := []struct {
		name      string
		sessionID string
		status    string
		wantErr   bool
	}{
		{"set idle", "s1", "idle", false},
		{"set active", "s1", "active", false},
		{"invalid status", "s1", "away", true},
		{"non-existent", "s99", "active", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.UpdatePresence(tt.sessionID, tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdatePresence() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHasVoted(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice")

	if r.HasVoted("s1") {
		t.Error("expected no vote initially")
	}
	r.CastVote("s1", "5")
	if !r.HasVoted("s1") {
		t.Error("expected vote after casting")
	}
	r.CastVote("s1", "")
	if r.HasVoted("s1") {
		t.Error("expected no vote after retract")
	}
	if r.HasVoted("ghost") {
		t.Error("expected false for non-existent participant")
	}
}

func TestActiveConnections(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice")
	r.Join("s2", "Bob")
	r.Join("s3", "Charlie")

	if got := r.ActiveConnections(); got != 3 {
		t.Errorf("expected 3 active, got %d", got)
	}

	r.Participants["s2"].Status = "disconnected"
	if got := r.ActiveConnections(); got != 2 {
		t.Errorf("expected 2 active, got %d", got)
	}
}

func TestTouchActivity(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	before := r.GetLastActivity()
	time.Sleep(1 * time.Millisecond)

	r.TouchActivity()

	after := r.GetLastActivity()
	if !after.After(before) {
		t.Error("TouchActivity should advance the timestamp")
	}
}

func TestGetLastActivity_ReasonableTime(t *testing.T) {
	now := time.Now()
	r, _ := NewRoom("room1", "Test", "")
	got := r.GetLastActivity()

	if got.Before(now.Add(-1*time.Second)) || got.After(now.Add(1*time.Second)) {
		t.Errorf("GetLastActivity() = %v, want within 1s of %v", got, now)
	}
}

func TestLastActivityUnixNano(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	nanos := r.LastActivityUnixNano()
	if nanos <= 0 {
		t.Errorf("LastActivityUnixNano() = %d, want > 0", nanos)
	}
}

func TestSetLastActivity(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	r.SetLastActivity(past)
	got := r.GetLastActivity()
	if !got.Equal(past) {
		t.Errorf("SetLastActivity/GetLastActivity roundtrip: got %v, want %v", got, past)
	}
}

func TestAllOperationsUpdateLastActivity(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice")

	ops := []struct {
		name string
		fn   func()
	}{
		{"Join", func() { r.Join("s2", "Bob") }},
		{"Leave", func() { r.Leave("s2") }},
		{"CastVote", func() { r.CastVote("s1", "5") }},
		{"Reveal", func() { r.Reveal() }},
		{"NewRound", func() { r.NewRound() }},
		{"ClearRoom", func() {
			r.ClearRoom()
			r.Join("s1", "Alice") // re-join for subsequent ops
		}},
		{"UpdateName", func() { r.UpdateName("s1", "Alicia") }},
		{"UpdatePresence", func() { r.UpdatePresence("s1", "idle") }},
	}

	for _, op := range ops {
		before := r.GetLastActivity()
		time.Sleep(1 * time.Millisecond)
		op.fn()
		after := r.GetLastActivity()
		if !after.After(before) {
			t.Errorf("%s: expected LastActivity to advance", op.name)
		}
	}
}
