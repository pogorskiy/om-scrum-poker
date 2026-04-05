package server

import (
	"encoding/json"
	"testing"
	"time"
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
