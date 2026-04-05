package server

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"om-scrum-poker/internal/domain"
)

// helper to drain one message from a client's send channel with a timeout.
func recvMessage(t *testing.T, c *Client, timeout time.Duration) *Envelope {
	t.Helper()
	select {
	case raw := <-c.send:
		var env Envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatalf("failed to unmarshal message: %v", err)
		}
		return &env
	case <-time.After(timeout):
		t.Fatal("timed out waiting for message")
		return nil
	}
}

// helper to assert no message is received within a short window.
func expectNoMessage(t *testing.T, c *Client, timeout time.Duration) {
	t.Helper()
	select {
	case msg := <-c.send:
		t.Fatalf("expected no message, got: %s", string(msg))
	case <-time.After(timeout):
		// Expected.
	}
}

func TestHandleUpdateName_NotJoined(t *testing.T) {
	rm := NewRoomManager()
	c := fakeClient("room-1", rm)
	// sessionID is empty — client has not joined yet.

	payload, _ := json.Marshal(UpdateNamePayload{UserName: "New Name"})
	handleUpdateName(c, rm, payload)

	env := recvMessage(t, c, 100*time.Millisecond)
	if env.Type != "error" {
		t.Fatalf("expected error message, got %q", env.Type)
	}
	var errPayload ErrorPayload
	json.Unmarshal(env.Payload, &errPayload)
	if errPayload.Code != "invalid_message" {
		t.Errorf("expected error code %q, got %q", "invalid_message", errPayload.Code)
	}
}

func TestHandleUpdateName_RoomNotFound(t *testing.T) {
	rm := NewRoomManager()
	c := fakeClient("nonexistent-room", rm)
	c.SetSessionID("sess-1")

	payload, _ := json.Marshal(UpdateNamePayload{UserName: "New Name"})
	handleUpdateName(c, rm, payload)

	env := recvMessage(t, c, 100*time.Millisecond)
	if env.Type != "error" {
		t.Fatalf("expected error message, got %q", env.Type)
	}
	var errPayload ErrorPayload
	json.Unmarshal(env.Payload, &errPayload)
	if errPayload.Code != "room_not_found" {
		t.Errorf("expected error code %q, got %q", "room_not_found", errPayload.Code)
	}
}

func TestHandleUpdateName_InvalidPayload(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreateRoom("room-1", "Test", "")
	c := fakeClient("room-1", rm)
	c.SetSessionID("sess-1")

	// Send malformed JSON as payload.
	handleUpdateName(c, rm, json.RawMessage(`{invalid json`))

	env := recvMessage(t, c, 100*time.Millisecond)
	if env.Type != "error" {
		t.Fatalf("expected error message, got %q", env.Type)
	}
	var errPayload ErrorPayload
	json.Unmarshal(env.Payload, &errPayload)
	if errPayload.Code != "invalid_name" {
		t.Errorf("expected error code %q, got %q", "invalid_name", errPayload.Code)
	}
}

func TestHandleUpdateName_EmptyUserName(t *testing.T) {
	rm := NewRoomManager()
	room, _ := rm.GetOrCreateRoom("room-1", "Test", "")
	room.Lock()
	room.Join("sess-1", "Alice", "")
	room.Unlock()

	c := fakeClient("room-1", rm)
	c.SetSessionID("sess-1")
	rm.RegisterClient("room-1", c)

	payload, _ := json.Marshal(UpdateNamePayload{UserName: ""})
	handleUpdateName(c, rm, payload)

	env := recvMessage(t, c, 100*time.Millisecond)
	if env.Type != "error" {
		t.Fatalf("expected error message, got %q", env.Type)
	}
	var errPayload ErrorPayload
	json.Unmarshal(env.Payload, &errPayload)
	if errPayload.Code != "invalid_name" {
		t.Errorf("expected error code %q, got %q", "invalid_name", errPayload.Code)
	}
}

func TestHandleUpdateName_Success_BroadcastsToAllClients(t *testing.T) {
	rm := NewRoomManager()
	room, _ := rm.GetOrCreateRoom("room-1", "Test", "")
	room.Lock()
	room.Join("sess-1", "Alice", "")
	room.Join("sess-2", "Bob", "")
	room.Unlock()

	c1 := fakeClient("room-1", rm)
	c1.SetSessionID("sess-1")
	c2 := fakeClient("room-1", rm)
	c2.SetSessionID("sess-2")
	rm.RegisterClient("room-1", c1)
	rm.RegisterClient("room-1", c2)

	payload, _ := json.Marshal(UpdateNamePayload{UserName: "Alice Updated"})
	handleUpdateName(c1, rm, payload)

	// Both clients should receive the name_updated broadcast.
	for _, c := range []*Client{c1, c2} {
		env := recvMessage(t, c, 100*time.Millisecond)
		if env.Type != "name_updated" {
			t.Fatalf("expected name_updated, got %q", env.Type)
		}
		var namePayload NameUpdatedPayload
		if err := json.Unmarshal(env.Payload, &namePayload); err != nil {
			t.Fatalf("failed to unmarshal NameUpdatedPayload: %v", err)
		}
		if namePayload.SessionID != "sess-1" {
			t.Errorf("expected sessionId %q, got %q", "sess-1", namePayload.SessionID)
		}
		if namePayload.UserName != "Alice Updated" {
			t.Errorf("expected userName %q, got %q", "Alice Updated", namePayload.UserName)
		}
	}
}

func TestHandleUpdateName_Success_UpdatesDomainModel(t *testing.T) {
	rm := NewRoomManager()
	room, _ := rm.GetOrCreateRoom("room-1", "Test", "")
	room.Lock()
	room.Join("sess-1", "Alice", "")
	room.Unlock()

	c := fakeClient("room-1", rm)
	c.SetSessionID("sess-1")
	rm.RegisterClient("room-1", c)

	payload, _ := json.Marshal(UpdateNamePayload{UserName: "Alice Renamed"})
	handleUpdateName(c, rm, payload)

	// Drain the broadcast message.
	recvMessage(t, c, 100*time.Millisecond)

	// Verify the domain model was updated.
	room.Lock()
	name := room.Participants["sess-1"].Name
	room.Unlock()

	if name != "Alice Renamed" {
		t.Errorf("expected participant name %q, got %q", "Alice Renamed", name)
	}
}

func TestHandleUpdateName_ParticipantNotInRoom(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreateRoom("room-1", "Test", "")
	// Room exists but participant "sess-ghost" is not in it.

	c := fakeClient("room-1", rm)
	c.SetSessionID("sess-ghost")

	payload, _ := json.Marshal(UpdateNamePayload{UserName: "Ghost"})
	handleUpdateName(c, rm, payload)

	env := recvMessage(t, c, 100*time.Millisecond)
	if env.Type != "error" {
		t.Fatalf("expected error message, got %q", env.Type)
	}
	var errPayload ErrorPayload
	json.Unmarshal(env.Payload, &errPayload)
	if errPayload.Code != "invalid_name" {
		t.Errorf("expected error code %q, got %q", "invalid_name", errPayload.Code)
	}
}

func TestHandleUpdateName_NoExtraMessagesOnError(t *testing.T) {
	rm := NewRoomManager()
	room, _ := rm.GetOrCreateRoom("room-1", "Test", "")
	room.Lock()
	room.Join("sess-1", "Alice", "")
	room.Unlock()

	c1 := fakeClient("room-1", rm)
	c1.SetSessionID("sess-1")
	c2 := fakeClient("room-1", rm)
	c2.SetSessionID("sess-2")
	rm.RegisterClient("room-1", c1)
	rm.RegisterClient("room-1", c2)

	// Send empty name — should error, not broadcast.
	payload, _ := json.Marshal(UpdateNamePayload{UserName: ""})
	handleUpdateName(c1, rm, payload)

	// c1 should get the error.
	env := recvMessage(t, c1, 100*time.Millisecond)
	if env.Type != "error" {
		t.Fatalf("expected error, got %q", env.Type)
	}

	// c2 should NOT receive anything.
	expectNoMessage(t, c2, 50*time.Millisecond)
}

// --- Disconnect handling tests ---

func TestHandleLeave_RemovesParticipant(t *testing.T) {
	rm := NewRoomManager()
	room, _ := rm.GetOrCreateRoom("room-1", "Test", "")
	room.Lock()
	room.Join("sess-1", "Alice", "")
	room.Join("sess-2", "Bob", "")
	room.Unlock()

	c1 := fakeClient("room-1", rm)
	c1.SetSessionID("sess-1")
	c2 := fakeClient("room-1", rm)
	c2.SetSessionID("sess-2")
	rm.RegisterClient("room-1", c1)
	rm.RegisterClient("room-1", c2)

	handleLeave(c1, rm)

	// Participant should be removed from room.
	room.Lock()
	_, exists := room.Participants["sess-1"]
	room.Unlock()
	if exists {
		t.Error("participant sess-1 should have been removed from room")
	}

	// c2 should receive participant_left broadcast.
	env := recvMessage(t, c2, 100*time.Millisecond)
	if env.Type != "participant_left" {
		t.Fatalf("expected participant_left, got %q", env.Type)
	}
	var leftPayload ParticipantLeftPayload
	json.Unmarshal(env.Payload, &leftPayload)
	if leftPayload.SessionID != "sess-1" {
		t.Errorf("expected sessionId %q, got %q", "sess-1", leftPayload.SessionID)
	}

	// c1 should NOT receive the broadcast (BroadcastExcept).
	expectNoMessage(t, c1, 50*time.Millisecond)
}

func TestHandleLeave_NotJoined(t *testing.T) {
	rm := NewRoomManager()
	room, _ := rm.GetOrCreateRoom("room-1", "Test", "")
	room.Lock()
	room.Join("sess-2", "Bob", "")
	room.Unlock()

	c := fakeClient("room-1", rm)
	// sessionID is empty — client has not joined.

	c2 := fakeClient("room-1", rm)
	c2.SetSessionID("sess-2")
	rm.RegisterClient("room-1", c2)

	// Should be a no-op, no panic, no broadcast.
	handleLeave(c, rm)

	expectNoMessage(t, c, 50*time.Millisecond)
	expectNoMessage(t, c2, 50*time.Millisecond)
}

func TestHandleLeave_RoomNotFound(t *testing.T) {
	rm := NewRoomManager()
	c := fakeClient("nonexistent", rm)
	c.SetSessionID("sess-1")

	// Should be a no-op, no panic.
	handleLeave(c, rm)

	expectNoMessage(t, c, 50*time.Millisecond)
}

func TestHandleLeave_UnregistersClient(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreateRoom("room-1", "Test", "")

	c := fakeClient("room-1", rm)
	c.SetSessionID("sess-1")
	rm.RegisterClient("room-1", c)

	if rm.ConnectionCount() != 1 {
		t.Fatalf("expected 1 connection before leave, got %d", rm.ConnectionCount())
	}

	handleLeave(c, rm)

	if rm.ConnectionCount() != 0 {
		t.Errorf("expected 0 connections after leave, got %d", rm.ConnectionCount())
	}
	// sessionID should be cleared after leave.
	if c.SessionID() != "" {
		t.Errorf("expected empty sessionID after leave, got %q", c.SessionID())
	}
}

// --- Reconnect (rejoin) handling tests ---

func TestHandleJoin_Rejoin_RestoresActiveStatus(t *testing.T) {
	rm := NewRoomManager()
	limiter := NewRateLimiter(DefaultRateLimitConfig())
	room, _ := rm.GetOrCreateRoom("room-1", "Test", "")

	const sid1 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa01"

	// First join.
	room.Lock()
	room.Join(sid1, "Alice", "")
	room.Unlock()

	// Simulate disconnect by marking participant as disconnected.
	room.Lock()
	room.Participants[sid1].Status = "disconnected"
	room.Unlock()

	// Set up a second client to observe broadcasts.
	c2 := fakeClient("room-1", rm)
	c2.SetSessionID("sess-2")
	rm.RegisterClient("room-1", c2)

	// Rejoin with a new client (same sessionID).
	c1 := fakeClient("room-1", rm)
	payload, _ := json.Marshal(JoinPayload{SessionID: sid1, UserName: "Alice"})
	handleJoin(c1, rm, limiter, "127.0.0.1", payload)

	// Drain room_state sent to c1.
	env := recvMessage(t, c1, 100*time.Millisecond)
	if env.Type != "room_state" {
		t.Fatalf("expected room_state, got %q", env.Type)
	}

	// c2 should receive presence_changed (not participant_joined, since it's a rejoin).
	env2 := recvMessage(t, c2, 100*time.Millisecond)
	if env2.Type != "presence_changed" {
		t.Fatalf("expected presence_changed on rejoin, got %q", env2.Type)
	}
	var presPayload PresenceChangedPayload
	json.Unmarshal(env2.Payload, &presPayload)
	if presPayload.SessionID != sid1 {
		t.Errorf("expected sessionId %q, got %q", sid1, presPayload.SessionID)
	}
	if presPayload.Status != "active" {
		t.Errorf("expected status %q, got %q", "active", presPayload.Status)
	}

	// Verify domain model.
	room.Lock()
	status := room.Participants[sid1].Status
	room.Unlock()
	if status != "active" {
		t.Errorf("expected participant status %q, got %q", "active", status)
	}
}

func TestHandleJoin_Rejoin_PreservesVote(t *testing.T) {
	rm := NewRoomManager()
	limiter := NewRateLimiter(DefaultRateLimitConfig())
	room, _ := rm.GetOrCreateRoom("room-1", "Test", "")

	const sid1 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa01"

	// Join and cast a vote.
	room.Lock()
	room.Join(sid1, "Alice", "")
	room.CastVote(sid1, "5")
	room.Unlock()

	// Simulate disconnect.
	room.Lock()
	room.Participants[sid1].Status = "disconnected"
	room.Unlock()

	// Rejoin.
	c := fakeClient("room-1", rm)
	payload, _ := json.Marshal(JoinPayload{SessionID: sid1, UserName: "Alice"})
	handleJoin(c, rm, limiter, "127.0.0.1", payload)

	// Drain room_state.
	recvMessage(t, c, 100*time.Millisecond)

	// Verify vote is preserved.
	room.Lock()
	vote := room.Participants[sid1].Vote
	room.Unlock()
	if vote != "5" {
		t.Errorf("expected vote %q to be preserved, got %q", "5", string(vote))
	}
}

func TestHandleJoin_Rejoin_SendsFullRoomState(t *testing.T) {
	rm := NewRoomManager()
	limiter := NewRateLimiter(DefaultRateLimitConfig())
	room, _ := rm.GetOrCreateRoom("room-1", "Test Room", "")

	const sid1 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa01"
	const sid2 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa02"

	// Set up existing participants.
	room.Lock()
	room.Join(sid1, "Alice", "")
	room.Join(sid2, "Bob", "")
	room.CastVote(sid2, "8")
	room.Participants[sid1].Status = "disconnected"
	room.Unlock()

	// Rejoin as sid1.
	c := fakeClient("room-1", rm)
	payload, _ := json.Marshal(JoinPayload{SessionID: sid1, UserName: "Alice"})
	handleJoin(c, rm, limiter, "127.0.0.1", payload)

	env := recvMessage(t, c, 100*time.Millisecond)
	if env.Type != "room_state" {
		t.Fatalf("expected room_state, got %q", env.Type)
	}

	var state RoomStatePayload
	if err := json.Unmarshal(env.Payload, &state); err != nil {
		t.Fatalf("failed to unmarshal room state: %v", err)
	}
	if state.RoomID != "room-1" {
		t.Errorf("expected roomId %q, got %q", "room-1", state.RoomID)
	}
	if state.RoomName != "Test Room" {
		t.Errorf("expected roomName %q, got %q", "Test Room", state.RoomName)
	}
	if len(state.Participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(state.Participants))
	}
	if state.Phase != domain.PhaseVoting {
		t.Errorf("expected phase %q, got %q", domain.PhaseVoting, state.Phase)
	}
}

// --- Presence lifecycle tests ---

func TestPresenceLifecycle_ActiveToDisconnectedToActive(t *testing.T) {
	rm := NewRoomManager()
	limiter := NewRateLimiter(DefaultRateLimitConfig())
	room, _ := rm.GetOrCreateRoom("room-1", "Test", "")

	const sid1 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa01"

	// Step 1: Join — participant becomes active.
	c1 := fakeClient("room-1", rm)
	payload, _ := json.Marshal(JoinPayload{SessionID: sid1, UserName: "Alice"})
	handleJoin(c1, rm, limiter, "127.0.0.1", payload)
	recvMessage(t, c1, 100*time.Millisecond) // drain room_state

	room.Lock()
	status := room.Participants[sid1].Status
	room.Unlock()
	if status != "active" {
		t.Fatalf("step 1: expected status %q, got %q", "active", status)
	}

	// Step 2: Simulate disconnect (as done in HandleWebSocket cleanup).
	room.Lock()
	room.Participants[sid1].Status = "disconnected"
	room.Unlock()

	room.Lock()
	status = room.Participants[sid1].Status
	room.Unlock()
	if status != "disconnected" {
		t.Fatalf("step 2: expected status %q, got %q", "disconnected", status)
	}

	// Step 3: Rejoin — status should be restored to active.
	c2 := fakeClient("room-1", rm)
	payload, _ = json.Marshal(JoinPayload{SessionID: sid1, UserName: "Alice"})
	handleJoin(c2, rm, limiter, "127.0.0.1", payload)
	recvMessage(t, c2, 100*time.Millisecond) // drain room_state

	room.Lock()
	status = room.Participants[sid1].Status
	room.Unlock()
	if status != "active" {
		t.Fatalf("step 3: expected status %q after rejoin, got %q", "active", status)
	}
}

func TestHandlePresence_InvalidStatus(t *testing.T) {
	rm := NewRoomManager()
	room, _ := rm.GetOrCreateRoom("room-1", "Test", "")
	room.Lock()
	room.Join("sess-1", "Alice", "")
	room.Unlock()

	c := fakeClient("room-1", rm)
	c.SetSessionID("sess-1")
	rm.RegisterClient("room-1", c)

	payload, _ := json.Marshal(PresencePayload{Status: "banana"})
	handlePresence(c, rm, payload)

	env := recvMessage(t, c, 100*time.Millisecond)
	if env.Type != "error" {
		t.Fatalf("expected error, got %q", env.Type)
	}
	var errPayload ErrorPayload
	json.Unmarshal(env.Payload, &errPayload)
	if errPayload.Code != "invalid_message" {
		t.Errorf("expected error code %q, got %q", "invalid_message", errPayload.Code)
	}
}

// --- Dispatch tests ---

func TestDispatch_LeaveEvent(t *testing.T) {
	rm := NewRoomManager()
	room, _ := rm.GetOrCreateRoom("room-1", "Test", "")
	room.Lock()
	room.Join("sess-1", "Alice", "")
	room.Unlock()

	c := fakeClient("room-1", rm)
	c.SetSessionID("sess-1")
	rm.RegisterClient("room-1", c)

	env := &Envelope{Type: "leave"}
	dispatch(context.Background(), c, rm, NewRateLimiter(DefaultRateLimitConfig()), "127.0.0.1", env)

	// After dispatch of leave, participant should be removed.
	room.Lock()
	_, exists := room.Participants["sess-1"]
	room.Unlock()
	if exists {
		t.Error("participant should have been removed by dispatch leave")
	}
	// Client should be unregistered.
	if rm.ConnectionCount() != 0 {
		t.Errorf("expected 0 connections after dispatch leave, got %d", rm.ConnectionCount())
	}
}

func TestDispatch_UnknownEvent(t *testing.T) {
	rm := NewRoomManager()
	c := fakeClient("room-1", rm)

	env := &Envelope{Type: "teleport"}
	dispatch(context.Background(), c, rm, NewRateLimiter(DefaultRateLimitConfig()), "127.0.0.1", env)

	msg := recvMessage(t, c, 100*time.Millisecond)
	if msg.Type != "error" {
		t.Fatalf("expected error for unknown event, got %q", msg.Type)
	}
	var errPayload ErrorPayload
	json.Unmarshal(msg.Payload, &errPayload)
	if errPayload.Code != "invalid_message" {
		t.Errorf("expected error code %q, got %q", "invalid_message", errPayload.Code)
	}
}

// TestClient_SessionID_ConcurrentAccess verifies that concurrent reads and writes
// to the client's sessionID do not cause a data race.
func TestClient_SessionID_ConcurrentAccess(t *testing.T) {
	rm := NewRoomManager()
	c := fakeClient("room-1", rm)

	var wg sync.WaitGroup
	const goroutines = 50

	// Half the goroutines write, half read.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		if i%2 == 0 {
			go func(id int) {
				defer wg.Done()
				c.SetSessionID("sess-" + string(rune('A'+id)))
			}(i)
		} else {
			go func() {
				defer wg.Done()
				_ = c.SessionID()
			}()
		}
	}

	wg.Wait()

	// Verify we can still read/write without issues.
	c.SetSessionID("final")
	if got := c.SessionID(); got != "final" {
		t.Errorf("expected sessionID %q, got %q", "final", got)
	}
}

// --- Room ID validation tests ---

func TestValidRoomID(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		valid bool
	}{
		{"lowercase letters", "myroom", true},
		{"digits", "12345", true},
		{"hyphens", "my-room-1", true},
		{"mixed valid", "a1-b2-c3", true},
		{"single char", "a", true},
		{"max length 64", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},
		{"empty", "", false},
		{"too long 65", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},
		{"uppercase", "MyRoom", false},
		{"spaces", "my room", false},
		{"special chars", "room@1", false},
		{"underscore", "room_1", false},
		{"slash", "room/1", false},
		{"dot", "room.1", false},
		{"unicode", "комната", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validRoomID.MatchString(tt.id)
			if got != tt.valid {
				t.Errorf("validRoomID(%q) = %v, want %v", tt.id, got, tt.valid)
			}
		})
	}
}

// --- Session ID validation tests ---

func TestValidSessionID(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		valid bool
	}{
		{"valid 32 hex", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa01", true},
		{"valid all digits", "00000000000000000000000000000000", true},
		{"valid all letters", "abcdefabcdefabcdefabcdefabcdefab", true},
		{"valid mixed", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4", true},
		{"empty", "", false},
		{"too short 31", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa0", false},
		{"too long 33", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa012", false},
		{"uppercase hex", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA01", false},
		{"mixed case", "AAAAAAAAAAAAAAAAaaaaaaaaaaaaaaaa", false},
		{"non-hex chars g", "gaaaaaaaaaaaaaaaaaaaaaaaaaaaaa01", false},
		{"non-hex chars z", "zaaaaaaaaaaaaaaaaaaaaaaaaaaaaa01", false},
		{"spaces", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa 1", false},
		{"hyphens", "aaaaaaaa-aaaaaaa-aaaaaaa-aaaaaaa", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validSessionID.MatchString(tt.id)
			if got != tt.valid {
				t.Errorf("validSessionID(%q) = %v, want %v", tt.id, got, tt.valid)
			}
		})
	}
}

// --- handleJoin session ID validation test ---

func TestHandleJoin_InvalidSessionIDFormat(t *testing.T) {
	rm := NewRoomManager()
	limiter := NewRateLimiter(DefaultRateLimitConfig())

	c := fakeClient("room-1", rm)
	payload, _ := json.Marshal(JoinPayload{SessionID: "not-valid-hex", UserName: "Alice"})
	handleJoin(c, rm, limiter, "127.0.0.1", payload)

	env := recvMessage(t, c, 100*time.Millisecond)
	if env.Type != "error" {
		t.Fatalf("expected error, got %q", env.Type)
	}
	var errPayload ErrorPayload
	json.Unmarshal(env.Payload, &errPayload)
	if errPayload.Code != "invalid_message" {
		t.Errorf("expected error code %q, got %q", "invalid_message", errPayload.Code)
	}
	if errPayload.Message != "invalid sessionId format" {
		t.Errorf("expected message %q, got %q", "invalid sessionId format", errPayload.Message)
	}
}

func TestHandleJoin_ValidSessionID(t *testing.T) {
	rm := NewRoomManager()
	limiter := NewRateLimiter(DefaultRateLimitConfig())

	c := fakeClient("room-1", rm)
	payload, _ := json.Marshal(JoinPayload{SessionID: "abcdef0123456789abcdef0123456789", UserName: "Alice"})
	handleJoin(c, rm, limiter, "127.0.0.1", payload)

	env := recvMessage(t, c, 100*time.Millisecond)
	if env.Type != "room_state" {
		t.Fatalf("expected room_state on valid join, got %q", env.Type)
	}
}

func TestHandleJoin_RoomNameFromPayload(t *testing.T) {
	rm := NewRoomManager()
	limiter := NewRateLimiter(DefaultRateLimitConfig())

	const sid = "abcdef0123456789abcdef0123456789"
	c := fakeClient("room-custom", rm)
	payload, _ := json.Marshal(JoinPayload{
		SessionID: sid,
		UserName:  "Alice",
		RoomName:  "Sprint 42",
	})
	handleJoin(c, rm, limiter, "127.0.0.1", payload)

	env := recvMessage(t, c, 100*time.Millisecond)
	if env.Type != "room_state" {
		t.Fatalf("expected room_state, got %q", env.Type)
	}

	var state RoomStatePayload
	if err := json.Unmarshal(env.Payload, &state); err != nil {
		t.Fatalf("unmarshal room state: %v", err)
	}
	if state.RoomName != "Sprint 42" {
		t.Errorf("expected roomName %q, got %q", "Sprint 42", state.RoomName)
	}
	if state.CreatedBy != "Alice" {
		t.Errorf("expected createdBy %q, got %q", "Alice", state.CreatedBy)
	}
}

func TestHandleJoin_RoomNameFallback(t *testing.T) {
	rm := NewRoomManager()
	limiter := NewRateLimiter(DefaultRateLimitConfig())

	const sid = "abcdef0123456789abcdef0123456789"
	c := fakeClient("room-fallback", rm)
	payload, _ := json.Marshal(JoinPayload{
		SessionID: sid,
		UserName:  "Alice",
	})
	handleJoin(c, rm, limiter, "127.0.0.1", payload)

	env := recvMessage(t, c, 100*time.Millisecond)
	if env.Type != "room_state" {
		t.Fatalf("expected room_state, got %q", env.Type)
	}

	var state RoomStatePayload
	if err := json.Unmarshal(env.Payload, &state); err != nil {
		t.Fatalf("unmarshal room state: %v", err)
	}
	if state.RoomName != "Alice's Room" {
		t.Errorf("expected roomName %q, got %q", "Alice's Room", state.RoomName)
	}
}

func TestHandleJoin_CreatedByOnlySetForFirstJoiner(t *testing.T) {
	rm := NewRoomManager()
	limiter := NewRateLimiter(DefaultRateLimitConfig())

	const sid1 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa01"
	const sid2 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa02"

	// First joiner creates the room.
	c1 := fakeClient("room-creator", rm)
	payload1, _ := json.Marshal(JoinPayload{
		SessionID: sid1,
		UserName:  "Alice",
		RoomName:  "Alice's Sprint",
	})
	handleJoin(c1, rm, limiter, "127.0.0.1", payload1)
	recvMessage(t, c1, 100*time.Millisecond) // drain room_state

	// Second joiner joins existing room.
	c2 := fakeClient("room-creator", rm)
	payload2, _ := json.Marshal(JoinPayload{
		SessionID: sid2,
		UserName:  "Bob",
		RoomName:  "Bob's Sprint",
	})
	handleJoin(c2, rm, limiter, "127.0.0.1", payload2)

	env := recvMessage(t, c2, 100*time.Millisecond)
	if env.Type != "room_state" {
		t.Fatalf("expected room_state, got %q", env.Type)
	}

	var state RoomStatePayload
	if err := json.Unmarshal(env.Payload, &state); err != nil {
		t.Fatalf("unmarshal room state: %v", err)
	}
	// Room name and creator should remain from the first joiner.
	if state.RoomName != "Alice's Sprint" {
		t.Errorf("expected roomName %q, got %q", "Alice's Sprint", state.RoomName)
	}
	if state.CreatedBy != "Alice" {
		t.Errorf("expected createdBy %q (first joiner), got %q", "Alice", state.CreatedBy)
	}
}

// --- Observer / Role tests ---

func TestHandleUpdateRole_NotJoined(t *testing.T) {
	rm := NewRoomManager()
	c := fakeClient("room-1", rm)

	payload, _ := json.Marshal(UpdateRolePayload{Role: "observer"})
	handleUpdateRole(c, rm, payload)

	env := recvMessage(t, c, 100*time.Millisecond)
	if env.Type != "error" {
		t.Fatalf("expected error, got %q", env.Type)
	}
	var errPayload ErrorPayload
	json.Unmarshal(env.Payload, &errPayload)
	if errPayload.Code != "invalid_message" {
		t.Errorf("expected error code %q, got %q", "invalid_message", errPayload.Code)
	}
}

func TestHandleUpdateRole_RoomNotFound(t *testing.T) {
	rm := NewRoomManager()
	c := fakeClient("nonexistent", rm)
	c.SetSessionID("sess-1")

	payload, _ := json.Marshal(UpdateRolePayload{Role: "observer"})
	handleUpdateRole(c, rm, payload)

	env := recvMessage(t, c, 100*time.Millisecond)
	if env.Type != "error" {
		t.Fatalf("expected error, got %q", env.Type)
	}
	var errPayload ErrorPayload
	json.Unmarshal(env.Payload, &errPayload)
	if errPayload.Code != "room_not_found" {
		t.Errorf("expected error code %q, got %q", "room_not_found", errPayload.Code)
	}
}

func TestHandleUpdateRole_InvalidPayload(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreateRoom("room-1", "Test", "")
	c := fakeClient("room-1", rm)
	c.SetSessionID("sess-1")

	handleUpdateRole(c, rm, json.RawMessage(`{invalid`))

	env := recvMessage(t, c, 100*time.Millisecond)
	if env.Type != "error" {
		t.Fatalf("expected error, got %q", env.Type)
	}
}

func TestHandleUpdateRole_InvalidRole(t *testing.T) {
	rm := NewRoomManager()
	room, _ := rm.GetOrCreateRoom("room-1", "Test", "")
	room.Lock()
	room.Join("sess-1", "Alice", "")
	room.Unlock()

	c := fakeClient("room-1", rm)
	c.SetSessionID("sess-1")
	rm.RegisterClient("room-1", c)

	payload, _ := json.Marshal(UpdateRolePayload{Role: "admin"})
	handleUpdateRole(c, rm, payload)

	env := recvMessage(t, c, 100*time.Millisecond)
	if env.Type != "error" {
		t.Fatalf("expected error, got %q", env.Type)
	}
}

func TestHandleUpdateRole_Success_BroadcastsToAll(t *testing.T) {
	rm := NewRoomManager()
	room, _ := rm.GetOrCreateRoom("room-1", "Test", "")
	room.Lock()
	room.Join("sess-1", "Alice", "")
	room.Join("sess-2", "Bob", "")
	room.Unlock()

	c1 := fakeClient("room-1", rm)
	c1.SetSessionID("sess-1")
	c2 := fakeClient("room-1", rm)
	c2.SetSessionID("sess-2")
	rm.RegisterClient("room-1", c1)
	rm.RegisterClient("room-1", c2)

	payload, _ := json.Marshal(UpdateRolePayload{Role: "observer"})
	handleUpdateRole(c1, rm, payload)

	// Both clients should receive role_updated.
	for _, c := range []*Client{c1, c2} {
		env := recvMessage(t, c, 100*time.Millisecond)
		if env.Type != "role_updated" {
			t.Fatalf("expected role_updated, got %q", env.Type)
		}
		var rolePayload RoleUpdatedPayload
		json.Unmarshal(env.Payload, &rolePayload)
		if rolePayload.SessionID != "sess-1" {
			t.Errorf("expected sessionId %q, got %q", "sess-1", rolePayload.SessionID)
		}
		if rolePayload.Role != "observer" {
			t.Errorf("expected role %q, got %q", "observer", rolePayload.Role)
		}
	}

	// Verify domain model.
	room.Lock()
	role := room.Participants["sess-1"].Role
	room.Unlock()
	if role != "observer" {
		t.Errorf("expected role %q in domain, got %q", "observer", role)
	}
}

func TestHandleJoin_WithObserverRole(t *testing.T) {
	rm := NewRoomManager()
	limiter := NewRateLimiter(DefaultRateLimitConfig())

	const sid = "abcdef0123456789abcdef0123456789"
	c := fakeClient("room-1", rm)
	payload, _ := json.Marshal(JoinPayload{
		SessionID: sid,
		UserName:  "Alice",
		Role:      "observer",
	})
	handleJoin(c, rm, limiter, "127.0.0.1", payload)

	env := recvMessage(t, c, 100*time.Millisecond)
	if env.Type != "room_state" {
		t.Fatalf("expected room_state, got %q", env.Type)
	}

	var state RoomStatePayload
	json.Unmarshal(env.Payload, &state)

	if len(state.Participants) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(state.Participants))
	}
	if state.Participants[0].Role != "observer" {
		t.Errorf("expected role %q, got %q", "observer", state.Participants[0].Role)
	}
}

func TestHandleJoin_DefaultVoterRole(t *testing.T) {
	rm := NewRoomManager()
	limiter := NewRateLimiter(DefaultRateLimitConfig())

	const sid = "abcdef0123456789abcdef0123456789"
	c := fakeClient("room-1", rm)
	payload, _ := json.Marshal(JoinPayload{
		SessionID: sid,
		UserName:  "Alice",
	})
	handleJoin(c, rm, limiter, "127.0.0.1", payload)

	env := recvMessage(t, c, 100*time.Millisecond)
	if env.Type != "room_state" {
		t.Fatalf("expected room_state, got %q", env.Type)
	}

	var state RoomStatePayload
	json.Unmarshal(env.Payload, &state)

	if len(state.Participants) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(state.Participants))
	}
	if state.Participants[0].Role != "voter" {
		t.Errorf("expected role %q, got %q", "voter", state.Participants[0].Role)
	}
}

func TestDispatch_UpdateRoleEvent(t *testing.T) {
	rm := NewRoomManager()
	room, _ := rm.GetOrCreateRoom("room-1", "Test", "")
	room.Lock()
	room.Join("sess-1", "Alice", "")
	room.Unlock()

	c := fakeClient("room-1", rm)
	c.SetSessionID("sess-1")
	rm.RegisterClient("room-1", c)

	payload, _ := json.Marshal(UpdateRolePayload{Role: "observer"})
	env := &Envelope{Type: "update_role", Payload: payload}
	dispatch(context.Background(), c, rm, NewRateLimiter(DefaultRateLimitConfig()), "127.0.0.1", env)

	msg := recvMessage(t, c, 100*time.Millisecond)
	if msg.Type != "role_updated" {
		t.Fatalf("expected role_updated, got %q", msg.Type)
	}
}
