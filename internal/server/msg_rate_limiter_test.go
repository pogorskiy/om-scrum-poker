package server

import (
	"testing"
	"time"
)

func TestMsgRateLimiter_AllowBurst(t *testing.T) {
	rl := NewMsgRateLimiter(5, 5)

	// Should allow up to capacity.
	for i := 0; i < 5; i++ {
		if !rl.Allow() {
			t.Fatalf("expected Allow() = true on message %d", i+1)
		}
	}

	// Next one should be denied.
	if rl.Allow() {
		t.Error("expected Allow() = false after burst exhausted")
	}
}

func TestMsgRateLimiter_Refill(t *testing.T) {
	rl := NewMsgRateLimiter(2, 10) // 2 burst, 10/sec refill

	// Exhaust tokens.
	rl.Allow()
	rl.Allow()
	if rl.Allow() {
		t.Error("expected Allow() = false after exhaustion")
	}

	// Wait for refill (at 10/sec, 100ms refills 1 token).
	time.Sleep(150 * time.Millisecond)

	if !rl.Allow() {
		t.Error("expected Allow() = true after refill")
	}
}

func TestMsgRateLimiter_DoesNotExceedCapacity(t *testing.T) {
	rl := NewMsgRateLimiter(3, 100) // high refill rate

	// Wait to accumulate many tokens.
	time.Sleep(100 * time.Millisecond)

	// Should allow exactly capacity, not more.
	allowed := 0
	for i := 0; i < 10; i++ {
		if rl.Allow() {
			allowed++
		}
	}
	if allowed != 3 {
		t.Errorf("expected exactly 3 allowed (capacity), got %d", allowed)
	}
}

func TestDefaultMsgRateLimiter(t *testing.T) {
	rl := DefaultMsgRateLimiter()

	// Should allow 20 messages in a burst.
	for i := 0; i < 20; i++ {
		if !rl.Allow() {
			t.Fatalf("expected Allow() = true on message %d", i+1)
		}
	}

	// 21st should be denied.
	if rl.Allow() {
		t.Error("expected Allow() = false after 20 burst messages")
	}
}

func TestMsgRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := NewMsgRateLimiter(100, 100)
	done := make(chan struct{})

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 20; j++ {
				rl.Allow()
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
	// No race condition — test passes if no panic.
}
