package server

import (
	"testing"
	"time"
)

func TestSend_BufferFull_ClosesClient(t *testing.T) {
	rm := NewRoomManager()
	c := fakeClient("room-1", rm)

	// Fill the send buffer completely.
	for i := 0; i < sendBufferSize; i++ {
		c.Send([]byte("msg"))
	}

	// Next send should trigger Close (close the done channel).
	c.Send([]byte("overflow"))

	// Verify the done channel is closed.
	select {
	case <-c.done:
		// Expected — client was closed.
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected done channel to be closed after send buffer overflow")
	}
}

func TestSend_BufferNotFull_DoesNotClose(t *testing.T) {
	rm := NewRoomManager()
	c := fakeClient("room-1", rm)

	c.Send([]byte("msg"))

	// Verify done channel is NOT closed.
	select {
	case <-c.done:
		t.Fatal("done channel should not be closed when buffer is not full")
	case <-time.After(50 * time.Millisecond):
		// Expected.
	}
}

func TestSend_BufferFull_DoubleClose_NoPanic(t *testing.T) {
	rm := NewRoomManager()
	c := fakeClient("room-1", rm)

	// Fill the buffer.
	for i := 0; i < sendBufferSize; i++ {
		c.Send([]byte("msg"))
	}

	// Multiple overflows should not panic (closeOnce protects).
	c.Send([]byte("overflow1"))
	c.Send([]byte("overflow2"))
	c.Send([]byte("overflow3"))

	// If we reach here without panic, test passes.
}
