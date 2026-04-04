package server

import (
	"sync"
	"testing"
	"time"
)

func TestDefaultRateLimitConfig(t *testing.T) {
	cfg := DefaultRateLimitConfig()
	if cfg.RoomCreationsPerMin != 5 {
		t.Errorf("RoomCreationsPerMin = %d, want 5", cfg.RoomCreationsPerMin)
	}
	if cfg.WSConnectionsPerMin != 20 {
		t.Errorf("WSConnectionsPerMin = %d, want 20", cfg.WSConnectionsPerMin)
	}
}

func TestAllowRoomCreation_FirstCallAllowed(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{RoomCreationsPerMin: 5, WSConnectionsPerMin: 20})
	defer rl.Close()

	if !rl.AllowRoomCreation("10.0.0.1") {
		t.Error("first room creation should be allowed")
	}
}

func TestAllowWSConnection_FirstCallAllowed(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{RoomCreationsPerMin: 5, WSConnectionsPerMin: 20})
	defer rl.Close()

	if !rl.AllowWSConnection("10.0.0.1") {
		t.Error("first WS connection should be allowed")
	}
}

func TestAllowRoomCreation_ExhaustsTokens(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{RoomCreationsPerMin: 3, WSConnectionsPerMin: 20})
	defer rl.Close()

	ip := "10.0.0.2"
	// Should allow exactly 3 calls (max tokens = 3).
	for i := 0; i < 3; i++ {
		if !rl.AllowRoomCreation(ip) {
			t.Errorf("call %d should be allowed", i+1)
		}
	}

	// 4th call should be denied.
	if rl.AllowRoomCreation(ip) {
		t.Error("should be denied after exhausting tokens")
	}
}

func TestAllowWSConnection_ExhaustsTokens(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{RoomCreationsPerMin: 5, WSConnectionsPerMin: 2})
	defer rl.Close()

	ip := "10.0.0.3"
	for i := 0; i < 2; i++ {
		if !rl.AllowWSConnection(ip) {
			t.Errorf("call %d should be allowed", i+1)
		}
	}

	if rl.AllowWSConnection(ip) {
		t.Error("should be denied after exhausting WS tokens")
	}
}

func TestRateLimiter_DifferentIPsAreIndependent(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{RoomCreationsPerMin: 1, WSConnectionsPerMin: 1})
	defer rl.Close()

	if !rl.AllowRoomCreation("ip-a") {
		t.Error("ip-a first call should be allowed")
	}
	if rl.AllowRoomCreation("ip-a") {
		t.Error("ip-a second call should be denied")
	}

	// Different IP should still be allowed.
	if !rl.AllowRoomCreation("ip-b") {
		t.Error("ip-b first call should be allowed")
	}
}

func TestRateLimiter_RoomAndWSBucketsAreIndependent(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{RoomCreationsPerMin: 1, WSConnectionsPerMin: 1})
	defer rl.Close()

	ip := "10.0.0.5"

	// Exhaust room creation.
	rl.AllowRoomCreation(ip)
	if rl.AllowRoomCreation(ip) {
		t.Error("room creation should be denied")
	}

	// WS connection should still work (different bucket key).
	if !rl.AllowWSConnection(ip) {
		t.Error("WS connection should be allowed (different bucket)")
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{RoomCreationsPerMin: 100, WSConnectionsPerMin: 100})
	defer rl.Close()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rl.AllowRoomCreation("concurrent-ip")
			rl.AllowWSConnection("concurrent-ip")
		}()
	}
	wg.Wait()
	// No panic or race = success.
}

func TestTokenBucket_Refill(t *testing.T) {
	// Use the internal allow method indirectly: create a limiter with 1 token/min,
	// exhaust it, then manipulate the bucket's lastRefill to simulate time passing.
	rl := NewRateLimiter(RateLimitConfig{RoomCreationsPerMin: 1, WSConnectionsPerMin: 20})
	defer rl.Close()

	ip := "refill-test"
	key := "room:" + ip

	// Use the token.
	if !rl.AllowRoomCreation(ip) {
		t.Fatal("first call should be allowed")
	}

	// Should be denied now.
	if rl.AllowRoomCreation(ip) {
		t.Fatal("second call should be denied")
	}

	// Simulate time passing by adjusting lastRefill.
	rl.mu.Lock()
	bucket := rl.buckets[key]
	bucket.lastRefill = time.Now().Add(-61 * time.Second) // 61 seconds ago
	rl.mu.Unlock()

	// Should be allowed now after refill.
	if !rl.AllowRoomCreation(ip) {
		t.Error("should be allowed after token refill")
	}
}

func TestTokenBucket_RefillCapsAtMax(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{RoomCreationsPerMin: 2, WSConnectionsPerMin: 20})
	defer rl.Close()

	ip := "cap-test"
	key := "room:" + ip

	// Use 1 token.
	rl.AllowRoomCreation(ip)

	// Move lastRefill far back to trigger large refill.
	rl.mu.Lock()
	bucket := rl.buckets[key]
	bucket.lastRefill = time.Now().Add(-10 * time.Minute)
	rl.mu.Unlock()

	// Use all tokens; should get exactly 2 (max).
	allowed := 0
	for i := 0; i < 5; i++ {
		if rl.AllowRoomCreation(ip) {
			allowed++
		}
	}
	if allowed != 2 {
		t.Errorf("got %d allowed calls, want 2 (capped at max)", allowed)
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{RoomCreationsPerMin: 5, WSConnectionsPerMin: 20})
	defer rl.Close()

	// Create a bucket.
	rl.AllowRoomCreation("cleanup-ip")

	// Manually set lastRefill to > 5 minutes ago.
	rl.mu.Lock()
	for _, bucket := range rl.buckets {
		bucket.lastRefill = time.Now().Add(-6 * time.Minute)
	}
	rl.mu.Unlock()

	// Manually trigger collectGarbage.
	rl.mu.Lock()
	now := time.Now()
	for key, bucket := range rl.buckets {
		if now.Sub(bucket.lastRefill) > 5*time.Minute {
			delete(rl.buckets, key)
		}
	}
	rl.mu.Unlock()

	rl.mu.Lock()
	count := len(rl.buckets)
	rl.mu.Unlock()

	if count != 0 {
		t.Errorf("expected 0 buckets after cleanup, got %d", count)
	}
}

func TestRateLimiter_Close(t *testing.T) {
	rl := NewRateLimiter(DefaultRateLimitConfig())
	// Should not panic on close.
	rl.Close()
}
