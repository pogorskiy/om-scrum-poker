package server

import (
	"sync"
	"testing"
	"time"

	"om-scrum-poker/internal/domain"
)

func TestNewRoomManager(t *testing.T) {
	rm := NewRoomManager()
	if rm.RoomCount() != 0 {
		t.Errorf("new manager should have 0 rooms, got %d", rm.RoomCount())
	}
	if rm.ConnectionCount() != 0 {
		t.Errorf("new manager should have 0 connections, got %d", rm.ConnectionCount())
	}
}

func TestGetOrCreateRoom_CreatesNew(t *testing.T) {
	rm := NewRoomManager()

	room, err := rm.GetOrCreateRoom("room-1", "Test Room", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room == nil {
		t.Fatal("room should not be nil")
	}
	if room.ID != "room-1" {
		t.Errorf("room ID = %q, want %q", room.ID, "room-1")
	}
	if room.Name != "Test Room" {
		t.Errorf("room Name = %q, want %q", room.Name, "Test Room")
	}
	if rm.RoomCount() != 1 {
		t.Errorf("room count = %d, want 1", rm.RoomCount())
	}
}

func TestGetOrCreateRoom_ReturnsExisting(t *testing.T) {
	rm := NewRoomManager()

	room1, _ := rm.GetOrCreateRoom("room-1", "First Name", "")
	room2, _ := rm.GetOrCreateRoom("room-1", "Second Name", "")

	if room1 != room2 {
		t.Error("should return the same room pointer")
	}
	// Name should not change on second call.
	if room2.Name != "First Name" {
		t.Errorf("name should remain %q, got %q", "First Name", room2.Name)
	}
	if rm.RoomCount() != 1 {
		t.Errorf("room count = %d, want 1", rm.RoomCount())
	}
}

func TestGetOrCreateRoom_EmptyID(t *testing.T) {
	rm := NewRoomManager()

	_, err := rm.GetOrCreateRoom("", "Name", "")
	if err == nil {
		t.Error("expected error for empty room ID")
	}
}

func TestGetOrCreateRoom_MultipleRooms(t *testing.T) {
	rm := NewRoomManager()

	rm.GetOrCreateRoom("a", "Room A", "")
	rm.GetOrCreateRoom("b", "Room B", "")
	rm.GetOrCreateRoom("c", "Room C", "")

	if rm.RoomCount() != 3 {
		t.Errorf("room count = %d, want 3", rm.RoomCount())
	}
}

func TestGetRoom_Exists(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreateRoom("room-1", "Test", "")

	room := rm.GetRoom("room-1")
	if room == nil {
		t.Fatal("expected room to exist")
	}
	if room.ID != "room-1" {
		t.Errorf("room ID = %q, want %q", room.ID, "room-1")
	}
}

func TestGetRoom_NotExists(t *testing.T) {
	rm := NewRoomManager()

	room := rm.GetRoom("nonexistent")
	if room != nil {
		t.Error("expected nil for nonexistent room")
	}
}

// fakeClient creates a minimal Client suitable for registration tests.
// It has a nil conn (we never use the WebSocket in these tests).
func fakeClient(roomID string, rm *RoomManager) *Client {
	return &Client{
		send:    make(chan []byte, sendBufferSize),
		roomID:  roomID,
		manager: rm,
		done:    make(chan struct{}),
	}
}

func TestRegisterClient(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreateRoom("room-1", "Test", "")

	c := fakeClient("room-1", rm)
	rm.RegisterClient("room-1", c)

	if rm.ConnectionCount() != 1 {
		t.Errorf("connection count = %d, want 1", rm.ConnectionCount())
	}
}

func TestRegisterClient_NoRoom(t *testing.T) {
	rm := NewRoomManager()

	// Should not panic even if room doesn't exist in rooms map.
	c := fakeClient("room-x", rm)
	rm.RegisterClient("room-x", c)

	if rm.ConnectionCount() != 1 {
		t.Errorf("connection count = %d, want 1", rm.ConnectionCount())
	}
}

func TestUnregisterClient(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreateRoom("room-1", "Test", "")

	c := fakeClient("room-1", rm)
	rm.RegisterClient("room-1", c)
	rm.UnregisterClient("room-1", c)

	if rm.ConnectionCount() != 0 {
		t.Errorf("connection count = %d, want 0", rm.ConnectionCount())
	}
}

func TestUnregisterClient_NotRegistered(t *testing.T) {
	rm := NewRoomManager()

	c := fakeClient("room-1", rm)
	// Should not panic.
	rm.UnregisterClient("room-1", c)
}

func TestUnregisterClient_WrongRoom(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreateRoom("room-1", "Test", "")

	c := fakeClient("room-1", rm)
	rm.RegisterClient("room-1", c)

	// Unregister from a different room.
	rm.UnregisterClient("room-2", c)

	// Should still be registered in room-1.
	if rm.ConnectionCount() != 1 {
		t.Errorf("connection count = %d, want 1", rm.ConnectionCount())
	}
}

func TestBroadcast(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreateRoom("room-1", "Test", "")

	c1 := fakeClient("room-1", rm)
	c2 := fakeClient("room-1", rm)
	rm.RegisterClient("room-1", c1)
	rm.RegisterClient("room-1", c2)

	msg := []byte(`{"type":"test"}`)
	rm.Broadcast("room-1", msg)

	// Both clients should receive the message.
	select {
	case got := <-c1.send:
		if string(got) != string(msg) {
			t.Errorf("c1 got %q, want %q", got, msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("c1 did not receive message")
	}

	select {
	case got := <-c2.send:
		if string(got) != string(msg) {
			t.Errorf("c2 got %q, want %q", got, msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("c2 did not receive message")
	}
}

func TestBroadcast_EmptyRoom(t *testing.T) {
	rm := NewRoomManager()
	// Should not panic on broadcast to nonexistent room.
	rm.Broadcast("nonexistent", []byte(`{}`))
}

func TestBroadcastExcept(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreateRoom("room-1", "Test", "")

	c1 := fakeClient("room-1", rm)
	c2 := fakeClient("room-1", rm)
	rm.RegisterClient("room-1", c1)
	rm.RegisterClient("room-1", c2)

	msg := []byte(`{"type":"test"}`)
	rm.BroadcastExcept("room-1", msg, c1)

	// c1 should NOT receive.
	select {
	case <-c1.send:
		t.Error("c1 should not have received the message")
	case <-time.After(50 * time.Millisecond):
		// Expected.
	}

	// c2 should receive.
	select {
	case got := <-c2.send:
		if string(got) != string(msg) {
			t.Errorf("c2 got %q, want %q", got, msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("c2 did not receive message")
	}
}

func TestBuildRoomState_Empty(t *testing.T) {
	rm := NewRoomManager()
	room, _ := rm.GetOrCreateRoom("room-1", "Test Room", "")

	room.Lock()
	state := rm.BuildRoomState(room)
	room.Unlock()

	if state.RoomID != "room-1" {
		t.Errorf("roomID = %q, want %q", state.RoomID, "room-1")
	}
	if state.RoomName != "Test Room" {
		t.Errorf("roomName = %q, want %q", state.RoomName, "Test Room")
	}
	if state.Phase != domain.PhaseVoting {
		t.Errorf("phase = %q, want %q", state.Phase, domain.PhaseVoting)
	}
	if len(state.Participants) != 0 {
		t.Errorf("expected 0 participants, got %d", len(state.Participants))
	}
	if state.Result != nil {
		t.Error("result should be nil in voting phase")
	}
}

func TestBuildRoomState_WithParticipants(t *testing.T) {
	rm := NewRoomManager()
	room, _ := rm.GetOrCreateRoom("room-1", "Test", "")

	room.Lock()
	room.Join("sess-b", "Bob")
	room.Join("sess-a", "Alice")
	room.CastVote("sess-a", "5")
	room.Unlock()

	room.Lock()
	state := rm.BuildRoomState(room)
	room.Unlock()

	if len(state.Participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(state.Participants))
	}

	// Should be sorted by session ID.
	if state.Participants[0].SessionID != "sess-a" {
		t.Errorf("first participant = %q, want %q", state.Participants[0].SessionID, "sess-a")
	}
	if state.Participants[1].SessionID != "sess-b" {
		t.Errorf("second participant = %q, want %q", state.Participants[1].SessionID, "sess-b")
	}

	// Alice voted.
	if !state.Participants[0].HasVoted {
		t.Error("Alice should have HasVoted = true")
	}
	// Bob did not vote.
	if state.Participants[1].HasVoted {
		t.Error("Bob should have HasVoted = false")
	}
}

func TestBuildRoomState_RevealPhase(t *testing.T) {
	rm := NewRoomManager()
	room, _ := rm.GetOrCreateRoom("room-1", "Test", "")

	room.Lock()
	room.Join("sess-a", "Alice")
	room.CastVote("sess-a", "5")
	room.Reveal()
	room.Unlock()

	room.Lock()
	state := rm.BuildRoomState(room)
	room.Unlock()

	if state.Phase != domain.PhaseReveal {
		t.Errorf("phase = %q, want %q", state.Phase, domain.PhaseReveal)
	}
	if state.Result == nil {
		t.Error("result should not be nil in reveal phase")
	}
}

func TestCollectGarbage_RemovesStaleEmptyRooms(t *testing.T) {
	rm := NewRoomManager()
	room, _ := rm.GetOrCreateRoom("stale-room", "Stale", "")

	// Make the room appear old.
	room.SetLastActivity(time.Now().Add(-25 * time.Hour))

	rm.collectGarbage()

	if rm.RoomCount() != 0 {
		t.Errorf("stale room should have been collected, room count = %d", rm.RoomCount())
	}
}

func TestCollectGarbage_KeepsRoomWithClients(t *testing.T) {
	rm := NewRoomManager()
	room, _ := rm.GetOrCreateRoom("active-room", "Active", "")

	// Add a client.
	c := fakeClient("active-room", rm)
	rm.RegisterClient("active-room", c)

	// Make the room appear old.
	room.SetLastActivity(time.Now().Add(-25 * time.Hour))

	rm.collectGarbage()

	if rm.RoomCount() != 1 {
		t.Error("room with connected clients should not be collected")
	}
}

func TestCollectGarbage_KeepsRecentRoom(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreateRoom("recent-room", "Recent", "")

	rm.collectGarbage()

	if rm.RoomCount() != 1 {
		t.Error("recent room should not be collected")
	}
}

func TestUptime(t *testing.T) {
	rm := NewRoomManager()
	time.Sleep(10 * time.Millisecond)

	uptime := rm.Uptime()
	if uptime < 10*time.Millisecond {
		t.Errorf("uptime should be at least 10ms, got %v", uptime)
	}
}

func TestConnectionCount_MultipleRooms(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreateRoom("r1", "R1", "")
	rm.GetOrCreateRoom("r2", "R2", "")

	rm.RegisterClient("r1", fakeClient("r1", rm))
	rm.RegisterClient("r1", fakeClient("r1", rm))
	rm.RegisterClient("r2", fakeClient("r2", rm))

	if rm.ConnectionCount() != 3 {
		t.Errorf("connection count = %d, want 3", rm.ConnectionCount())
	}
}

func TestRoomManager_ConcurrentAccess(t *testing.T) {
	rm := NewRoomManager()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			roomID := "room-concurrent"
			rm.GetOrCreateRoom(roomID, "Test", "")
			c := fakeClient(roomID, rm)
			rm.RegisterClient(roomID, c)
			rm.Broadcast(roomID, []byte(`{"type":"ping"}`))
			rm.GetRoom(roomID)
			rm.RoomCount()
			rm.ConnectionCount()
			rm.UnregisterClient(roomID, c)
		}(i)
	}
	wg.Wait()
	// No panic or race = success.
}

func TestUpdatePingTime(t *testing.T) {
	rm := NewRoomManager()
	room, _ := rm.GetOrCreateRoom("room-1", "Test", "")

	room.Lock()
	room.Join("sess-1", "Alice")
	room.Unlock()

	before := time.Now()
	time.Sleep(1 * time.Millisecond)
	rm.UpdatePingTime("room-1", "sess-1")

	room.Lock()
	p := room.Participants["sess-1"]
	lastPing := p.LastPing
	room.Unlock()

	if lastPing.Before(before) {
		t.Error("lastPing should have been updated")
	}
}

func TestUpdatePingTime_NonexistentRoom(t *testing.T) {
	rm := NewRoomManager()
	// Should not panic.
	rm.UpdatePingTime("nonexistent", "sess-1")
}

func TestUpdatePingTime_NonexistentParticipant(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreateRoom("room-1", "Test", "")
	// Should not panic.
	rm.UpdatePingTime("room-1", "nonexistent")
}

func TestStartGC(t *testing.T) {
	rm := NewRoomManager()
	stop := rm.StartGC()
	// Should not panic on stop.
	stop()
}
