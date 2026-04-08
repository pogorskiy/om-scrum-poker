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
			p, isNew, err := r.Join(tt.sessionID, tt.userName, "")
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
	r.Join("s1", "Alice", "")

	// Simulate disconnect.
	r.Participants["s1"].Status = "disconnected"

	p, isNew, err := r.Join("s1", "Alice", "")
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
		_, _, err := r.Join(string(rune('A'+i))+"id", "User", "")
		if err != nil {
			t.Fatalf("unexpected error filling room: %v", err)
		}
	}
	_, _, err := r.Join("overflow", "Overflow", "")
	if err == nil {
		t.Fatal("expected room_full error")
	}
}

func TestLeave(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice", "")

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
	r.Join("s1", "Alice", "")

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
	r.Join("s1", "Alice", "")
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
	r.Join("s1", "Alice", "")
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
	r.Join("s1", "Alice", "")
	r.Join("s2", "Bob", "")
	r.CastVote("s1", "5")
	r.CastVote("s2", "8")
	r.Reveal()

	if err := r.NewRound(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.Phase != PhaseVoting {
		t.Errorf("expected phase %q, got %q", PhaseVoting, r.Phase)
	}
	for _, p := range r.Participants {
		if p.Vote != "" {
			t.Errorf("expected empty vote after new round, got %q for %s", p.Vote, p.Name)
		}
	}
}

func TestNewRound_AlreadyVoting(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice", "")
	// Phase is already voting — NewRound should return error.
	err := r.NewRound()
	if err == nil {
		t.Fatal("expected error when calling NewRound in voting phase")
	}
	if err.Error() != "already in voting phase" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReveal_AlreadyRevealed(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice", "")
	r.CastVote("s1", "5")
	r.Reveal()

	// Second reveal should fail.
	_, err := r.Reveal()
	if err == nil {
		t.Fatal("expected error when calling Reveal in reveal phase")
	}
	if err.Error() != "already in reveal phase" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClearRoom(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice", "")
	r.Join("s2", "Bob", "")
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
	r.Join("s1", "Alice", "")

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
	r.Join("s1", "Alice", "")

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
	r.Join("s1", "Alice", "")

	longName := "ABCDEFGHIJKLMNOPQRSTUVWXYZ12345678" // 34 chars, exceeds MaxParticipantName (30)
	err := r.UpdateName("s1", longName)
	if err != nil {
		t.Fatalf("UpdateName() unexpected error: %v", err)
	}
	got := r.Participants["s1"].Name
	runes := []rune(got)
	if len(runes) != MaxParticipantName {
		t.Errorf("expected name length %d runes, got %d", MaxParticipantName, len(runes))
	}
	expected := string([]rune(longName)[:MaxParticipantName])
	if got != expected {
		t.Errorf("expected truncated name %q, got %q", expected, got)
	}
}

func TestUpdateName_EmojiTruncation(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice", "")

	// 31 runes: emoji should not be split mid-character.
	emojiName := "Hello🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉"
	err := r.UpdateName("s1", emojiName)
	if err != nil {
		t.Fatalf("UpdateName() unexpected error: %v", err)
	}
	got := r.Participants["s1"].Name
	runes := []rune(got)
	if len(runes) > MaxParticipantName {
		t.Errorf("expected at most %d runes, got %d", MaxParticipantName, len(runes))
	}
	// Verify the result is valid UTF-8 (Go strings are valid UTF-8 by construction
	// when built from []rune, but let's be explicit).
	for i, r := range got {
		if r == '\uFFFD' {
			t.Errorf("invalid UTF-8 at byte offset %d", i)
		}
	}
}

func TestSanitizeName_ZalgoText(t *testing.T) {
	// Zalgo text: base char + many combining marks.
	zalgo := "t\u0300\u0301\u0302\u0303\u0304\u0305e\u0300\u0301\u0302\u0303\u0304\u0305s\u0300\u0301\u0302\u0303\u0304\u0305t"
	got := sanitizeName(zalgo)

	// Each base char should keep at most maxCombiningMarks combining marks.
	combiningCount := 0
	for _, r := range got {
		if r >= 0x0300 && r <= 0x036F { // combining diacritical marks block
			combiningCount++
		} else {
			combiningCount = 0
		}
		if combiningCount > maxCombiningMarks {
			t.Errorf("too many consecutive combining marks in sanitized name %q", got)
			break
		}
	}
}

func TestUpdateName_UpdatesLastActivity(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice", "")

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
	r.Join("s1", "Alice", "")

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
	r.Join("s1", "Alice", "")

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
	r.Join("s1", "Alice", "")
	r.Join("s2", "Bob", "")
	r.Join("s3", "Charlie", "")

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
	r.Join("s1", "Alice", "")

	ops := []struct {
		name string
		fn   func()
	}{
		{"Join", func() { r.Join("s2", "Bob", "") }},
		{"Leave", func() { r.Leave("s2") }},
		{"CastVote", func() { r.CastVote("s1", "5") }},
		{"Reveal", func() { r.Reveal() }},
		{"NewRound", func() { r.NewRound() }}, //nolint: errcheck — activity test
		{"ClearRoom", func() {
			r.ClearRoom()
			r.Join("s1", "Alice", "") // re-join for subsequent ops
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

func TestObserverCannotVote(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice", "observer")

	err := r.CastVote("s1", "5")
	if err == nil {
		t.Fatal("expected error when observer tries to vote")
	}
	if r.Participants["s1"].Vote != "" {
		t.Error("observer should not have a vote")
	}
}

func TestJoinWithObserverRole(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")

	p, isNew, err := r.Join("s1", "Alice", "observer")
	if err != nil {
		t.Fatal(err)
	}
	if !isNew {
		t.Error("expected new participant")
	}
	if p.Role != "observer" {
		t.Errorf("expected role %q, got %q", "observer", p.Role)
	}
}

func TestJoinWithEmptyRoleDefaultsToVoter(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")

	p, _, err := r.Join("s1", "Alice", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Role != "voter" {
		t.Errorf("expected role %q, got %q", "voter", p.Role)
	}
}

func TestJoinWithInvalidRole(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")

	_, _, err := r.Join("s1", "Alice", "admin")
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestUpdateRole(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice", "voter")
	r.CastVote("s1", "5")

	// Switch to observer — should clear vote.
	err := r.UpdateRole("s1", "observer")
	if err != nil {
		t.Fatal(err)
	}
	if r.Participants["s1"].Role != "observer" {
		t.Errorf("expected role %q, got %q", "observer", r.Participants["s1"].Role)
	}
	if r.Participants["s1"].Vote != "" {
		t.Error("expected vote to be cleared when switching to observer")
	}

	// Switch back to voter.
	err = r.UpdateRole("s1", "voter")
	if err != nil {
		t.Fatal(err)
	}
	if r.Participants["s1"].Role != "voter" {
		t.Errorf("expected role %q, got %q", "voter", r.Participants["s1"].Role)
	}
}

func TestUpdateRole_InvalidRole(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice", "voter")

	err := r.UpdateRole("s1", "admin")
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestUpdateRole_NonExistent(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")

	err := r.UpdateRole("ghost", "voter")
	if err == nil {
		t.Fatal("expected error for non-existent participant")
	}
}

func TestRejoinUpdatesRole(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice", "voter")
	r.CastVote("s1", "5")

	// Rejoin as observer — should update role and clear vote.
	p, isNew, err := r.Join("s1", "Alice", "observer")
	if err != nil {
		t.Fatal(err)
	}
	if isNew {
		t.Error("expected rejoin, got new")
	}
	if p.Role != "observer" {
		t.Errorf("expected role %q, got %q", "observer", p.Role)
	}
	if p.Vote != "" {
		t.Error("expected vote to be cleared on rejoin as observer")
	}
}

func TestTimer_SetDuration_Valid(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")

	for _, d := range []int{30, 60, 300, 600} {
		// Reset to idle first if needed.
		r.Timer.State = TimerIdle
		if err := r.SetTimerDuration(d); err != nil {
			t.Errorf("SetTimerDuration(%d) unexpected error: %v", d, err)
		}
		if r.Timer.Duration != d {
			t.Errorf("expected duration %d, got %d", d, r.Timer.Duration)
		}
	}
}

func TestTimer_SetDuration_OutOfRange(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")

	for _, d := range []int{0, 29, 601} {
		if err := r.SetTimerDuration(d); err == nil {
			t.Errorf("SetTimerDuration(%d) expected error, got nil", d)
		}
	}
}

func TestTimer_SetDuration_NotIdle(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.StartTimer()

	if err := r.SetTimerDuration(60); err == nil {
		t.Error("expected error when setting duration while running")
	}
}

func TestTimer_Start_FromIdle(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")

	changed := r.StartTimer()
	if !changed {
		t.Error("expected StartTimer to return true")
	}
	if r.Timer.State != TimerRunning {
		t.Errorf("expected state %q, got %q", TimerRunning, r.Timer.State)
	}
	if r.Timer.StartedAt.IsZero() {
		t.Error("expected StartedAt to be set")
	}
}

func TestTimer_Start_AlreadyRunning(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.StartTimer()

	changed := r.StartTimer()
	if changed {
		t.Error("expected StartTimer to return false when already running")
	}
}

func TestTimer_Start_FromExpired(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Timer.State = TimerExpired

	changed := r.StartTimer()
	if changed {
		t.Error("expected StartTimer to return false from expired state")
	}
	if r.Timer.State != TimerExpired {
		t.Errorf("expected state to remain %q, got %q", TimerExpired, r.Timer.State)
	}
}

func TestTimer_Reset_FromRunning(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.StartTimer()

	changed := r.ResetTimer()
	if !changed {
		t.Error("expected ResetTimer to return true")
	}
	if r.Timer.State != TimerIdle {
		t.Errorf("expected state %q, got %q", TimerIdle, r.Timer.State)
	}
	if !r.Timer.StartedAt.IsZero() {
		t.Error("expected StartedAt to be zero after reset")
	}
}

func TestTimer_Reset_AlreadyIdle(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")

	changed := r.ResetTimer()
	if changed {
		t.Error("expected ResetTimer to return false when already idle")
	}
}

func TestTimer_Info_Idle(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.SetTimerDuration(60)

	snap := r.TimerInfo()
	if snap.State != TimerIdle {
		t.Errorf("expected state %q, got %q", TimerIdle, snap.State)
	}
	if snap.Remaining != 60 {
		t.Errorf("expected remaining 60, got %d", snap.Remaining)
	}
}

func TestTimer_Info_Running(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.SetTimerDuration(60)
	r.StartTimer()

	// Immediately after start, remaining should be close to duration.
	snap := r.TimerInfo()
	if snap.State != TimerRunning {
		t.Errorf("expected state %q, got %q", TimerRunning, snap.State)
	}
	if snap.Remaining < 59 || snap.Remaining > 60 {
		t.Errorf("expected remaining ~60, got %d", snap.Remaining)
	}
}

func TestTimer_Info_AutoExpire(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Timer.State = TimerRunning
	r.Timer.StartedAt = time.Now().Add(-time.Duration(r.Timer.Duration+1) * time.Second)

	snap := r.TimerInfo()
	if snap.State != TimerExpired {
		t.Errorf("expected state %q, got %q", TimerExpired, snap.State)
	}
	if snap.Remaining != 0 {
		t.Errorf("expected remaining 0, got %d", snap.Remaining)
	}
	// Verify the room's timer state was also updated.
	if r.Timer.State != TimerExpired {
		t.Errorf("expected room timer state %q, got %q", TimerExpired, r.Timer.State)
	}
}

func TestNewRound_ResetsTimer(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice", "")
	r.CastVote("s1", "5")
	r.Reveal()

	// Start timer and set a non-default duration.
	r.SetTimerDuration(120)
	r.StartTimer()

	if err := r.NewRound(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.Timer.State != TimerIdle {
		t.Errorf("expected timer state %q after NewRound, got %q", TimerIdle, r.Timer.State)
	}
	if !r.Timer.StartedAt.IsZero() {
		t.Error("expected StartedAt to be zero after NewRound")
	}
	// Duration should be preserved.
	if r.Timer.Duration != 120 {
		t.Errorf("expected timer duration 120 after NewRound, got %d", r.Timer.Duration)
	}
}

func TestClearRoom_ResetsTimerToDefaults(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice", "")
	r.SetTimerDuration(120)
	r.StartTimer()

	r.ClearRoom()

	if r.Timer.State != TimerIdle {
		t.Errorf("expected timer state %q after ClearRoom, got %q", TimerIdle, r.Timer.State)
	}
	if r.Timer.Duration != DefaultTimerDuration {
		t.Errorf("expected timer duration %d after ClearRoom, got %d", DefaultTimerDuration, r.Timer.Duration)
	}
	if !r.Timer.StartedAt.IsZero() {
		t.Error("expected StartedAt to be zero after ClearRoom")
	}
}
