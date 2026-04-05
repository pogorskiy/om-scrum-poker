package server

import (
	"context"
	"encoding/json"
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
	c.sessionID = "sess-1"

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
	rm.GetOrCreateRoom("room-1", "Test")
	c := fakeClient("room-1", rm)
	c.sessionID = "sess-1"

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
	room, _ := rm.GetOrCreateRoom("room-1", "Test")
	room.Lock()
	room.Join("sess-1", "Alice")
	room.Unlock()

	c := fakeClient("room-1", rm)
	c.sessionID = "sess-1"
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
	room, _ := rm.GetOrCreateRoom("room-1", "Test")
	room.Lock()
	room.Join("sess-1", "Alice")
	room.Join("sess-2", "Bob")
	room.Unlock()

	c1 := fakeClient("room-1", rm)
	c1.sessionID = "sess-1"
	c2 := fakeClient("room-1", rm)
	c2.sessionID = "sess-2"
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
	room, _ := rm.GetOrCreateRoom("room-1", "Test")
	room.Lock()
	room.Join("sess-1", "Alice")
	room.Unlock()

	c := fakeClient("room-1", rm)
	c.sessionID = "sess-1"
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
	rm.GetOrCreateRoom("room-1", "Test")
	// Room exists but participant "sess-ghost" is not in it.

	c := fakeClient("room-1", rm)
	c.sessionID = "sess-ghost"

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
	room, _ := rm.GetOrCreateRoom("room-1", "Test")
	room.Lock()
	room.Join("sess-1", "Alice")
	room.Unlock()

	c1 := fakeClient("room-1", rm)
	c1.sessionID = "sess-1"
	c2 := fakeClient("room-1", rm)
	c2.sessionID = "sess-2"
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
	room, _ := rm.GetOrCreateRoom("room-1", "Test")
	room.Lock()
	room.Join("sess-1", "Alice")
	room.Join("sess-2", "Bob")
	room.Unlock()

	c1 := fakeClient("room-1", rm)
	c1.sessionID = "sess-1"
	c2 := fakeClient("room-1", rm)
	c2.sessionID = "sess-2"
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
	room, _ := rm.GetOrCreateRoom("room-1", "Test")
	room.Lock()
	room.Join("sess-2", "Bob")
	room.Unlock()

	c := fakeClient("room-1", rm)
	// sessionID is empty — client has not joined.

	c2 := fakeClient("room-1", rm)
	c2.sessionID = "sess-2"
	rm.RegisterClient("room-1", c2)

	// Should be a no-op, no panic, no broadcast.
	handleLeave(c, rm)

	expectNoMessage(t, c, 50*time.Millisecond)
	expectNoMessage(t, c2, 50*time.Millisecond)
}

func TestHandleLeave_RoomNotFound(t *testing.T) {
	rm := NewRoomManager()
	c := fakeClient("nonexistent", rm)
	c.sessionID = "sess-1"

	// Should be a no-op, no panic.
	handleLeave(c, rm)

	expectNoMessage(t, c, 50*time.Millisecond)
}

func TestHandleLeave_UnregistersClient(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreateRoom("room-1", "Test")

	c := fakeClient("room-1", rm)
	c.sessionID = "sess-1"
	rm.RegisterClient("room-1", c)

	if rm.ConnectionCount() != 1 {
		t.Fatalf("expected 1 connection before leave, got %d", rm.ConnectionCount())
	}

	handleLeave(c, rm)

	if rm.ConnectionCount() != 0 {
		t.Errorf("expected 0 connections after leave, got %d", rm.ConnectionCount())
	}
	// sessionID should be cleared after leave.
	if c.sessionID != "" {
		t.Errorf("expected empty sessionID after leave, got %q", c.sessionID)
	}
}

// --- Reconnect (rejoin) handling tests ---

func TestHandleJoin_Rejoin_RestoresActiveStatus(t *testing.T) {
	rm := NewRoomManager()
	limiter := NewRateLimiter(DefaultRateLimitConfig())
	room, _ := rm.GetOrCreateRoom("room-1", "Test")

	// First join.
	room.Lock()
	room.Join("sess-1", "Alice")
	room.Unlock()

	// Simulate disconnect by marking participant as disconnected.
	room.Lock()
	room.Participants["sess-1"].Status = "disconnected"
	room.Unlock()

	// Set up a second client to observe broadcasts.
	c2 := fakeClient("room-1", rm)
	c2.sessionID = "sess-2"
	rm.RegisterClient("room-1", c2)

	// Rejoin with a new client (same sessionID).
	c1 := fakeClient("room-1", rm)
	payload, _ := json.Marshal(JoinPayload{SessionID: "sess-1", UserName: "Alice"})
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
	if presPayload.SessionID != "sess-1" {
		t.Errorf("expected sessionId %q, got %q", "sess-1", presPayload.SessionID)
	}
	if presPayload.Status != "active" {
		t.Errorf("expected status %q, got %q", "active", presPayload.Status)
	}

	// Verify domain model.
	room.Lock()
	status := room.Participants["sess-1"].Status
	room.Unlock()
	if status != "active" {
		t.Errorf("expected participant status %q, got %q", "active", status)
	}
}

func TestHandleJoin_Rejoin_PreservesVote(t *testing.T) {
	rm := NewRoomManager()
	limiter := NewRateLimiter(DefaultRateLimitConfig())
	room, _ := rm.GetOrCreateRoom("room-1", "Test")

	// Join and cast a vote.
	room.Lock()
	room.Join("sess-1", "Alice")
	room.CastVote("sess-1", "5")
	room.Unlock()

	// Simulate disconnect.
	room.Lock()
	room.Participants["sess-1"].Status = "disconnected"
	room.Unlock()

	// Rejoin.
	c := fakeClient("room-1", rm)
	payload, _ := json.Marshal(JoinPayload{SessionID: "sess-1", UserName: "Alice"})
	handleJoin(c, rm, limiter, "127.0.0.1", payload)

	// Drain room_state.
	recvMessage(t, c, 100*time.Millisecond)

	// Verify vote is preserved.
	room.Lock()
	vote := room.Participants["sess-1"].Vote
	room.Unlock()
	if vote != "5" {
		t.Errorf("expected vote %q to be preserved, got %q", "5", string(vote))
	}
}

func TestHandleJoin_Rejoin_SendsFullRoomState(t *testing.T) {
	rm := NewRoomManager()
	limiter := NewRateLimiter(DefaultRateLimitConfig())
	room, _ := rm.GetOrCreateRoom("room-1", "Test Room")

	// Set up existing participants.
	room.Lock()
	room.Join("sess-1", "Alice")
	room.Join("sess-2", "Bob")
	room.CastVote("sess-2", "8")
	room.Participants["sess-1"].Status = "disconnected"
	room.Unlock()

	// Rejoin as sess-1.
	c := fakeClient("room-1", rm)
	payload, _ := json.Marshal(JoinPayload{SessionID: "sess-1", UserName: "Alice"})
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
	room, _ := rm.GetOrCreateRoom("room-1", "Test")

	// Step 1: Join — participant becomes active.
	c1 := fakeClient("room-1", rm)
	payload, _ := json.Marshal(JoinPayload{SessionID: "sess-1", UserName: "Alice"})
	handleJoin(c1, rm, limiter, "127.0.0.1", payload)
	recvMessage(t, c1, 100*time.Millisecond) // drain room_state

	room.Lock()
	status := room.Participants["sess-1"].Status
	room.Unlock()
	if status != "active" {
		t.Fatalf("step 1: expected status %q, got %q", "active", status)
	}

	// Step 2: Simulate disconnect (as done in HandleWebSocket cleanup).
	room.Lock()
	room.Participants["sess-1"].Status = "disconnected"
	room.Unlock()

	room.Lock()
	status = room.Participants["sess-1"].Status
	room.Unlock()
	if status != "disconnected" {
		t.Fatalf("step 2: expected status %q, got %q", "disconnected", status)
	}

	// Step 3: Rejoin — status should be restored to active.
	c2 := fakeClient("room-1", rm)
	payload, _ = json.Marshal(JoinPayload{SessionID: "sess-1", UserName: "Alice"})
	handleJoin(c2, rm, limiter, "127.0.0.1", payload)
	recvMessage(t, c2, 100*time.Millisecond) // drain room_state

	room.Lock()
	status = room.Participants["sess-1"].Status
	room.Unlock()
	if status != "active" {
		t.Fatalf("step 3: expected status %q after rejoin, got %q", "active", status)
	}
}

func TestHandlePresence_InvalidStatus(t *testing.T) {
	rm := NewRoomManager()
	room, _ := rm.GetOrCreateRoom("room-1", "Test")
	room.Lock()
	room.Join("sess-1", "Alice")
	room.Unlock()

	c := fakeClient("room-1", rm)
	c.sessionID = "sess-1"
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
	room, _ := rm.GetOrCreateRoom("room-1", "Test")
	room.Lock()
	room.Join("sess-1", "Alice")
	room.Unlock()

	c := fakeClient("room-1", rm)
	c.sessionID = "sess-1"
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
