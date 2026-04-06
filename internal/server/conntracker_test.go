package server

import (
	"fmt"
	"sync"
	"testing"
)

func TestConnTracker_DefaultConfig(t *testing.T) {
	cfg := DefaultConnTrackerConfig()
	if cfg.MaxPerIP != 100 {
		t.Errorf("MaxPerIP = %d, want 100", cfg.MaxPerIP)
	}
	if cfg.MaxTotal != 1000 {
		t.Errorf("MaxTotal = %d, want 1000", cfg.MaxTotal)
	}
}

func TestConnTracker_BasicAddRemove(t *testing.T) {
	ct := NewConnTracker(ConnTrackerConfig{MaxPerIP: 10, MaxTotal: 100})

	if !ct.TryAdd("10.0.0.1") {
		t.Fatal("first add should succeed")
	}
	if ct.ActivePerIP("10.0.0.1") != 1 {
		t.Errorf("ActivePerIP = %d, want 1", ct.ActivePerIP("10.0.0.1"))
	}
	if ct.ActiveTotal() != 1 {
		t.Errorf("ActiveTotal = %d, want 1", ct.ActiveTotal())
	}

	ct.Remove("10.0.0.1")
	if ct.ActivePerIP("10.0.0.1") != 0 {
		t.Errorf("ActivePerIP after remove = %d, want 0", ct.ActivePerIP("10.0.0.1"))
	}
	if ct.ActiveTotal() != 0 {
		t.Errorf("ActiveTotal after remove = %d, want 0", ct.ActiveTotal())
	}
}

func TestConnTracker_PerIPLimit(t *testing.T) {
	ct := NewConnTracker(ConnTrackerConfig{MaxPerIP: 3, MaxTotal: 100})

	for i := 0; i < 3; i++ {
		if !ct.TryAdd("10.0.0.1") {
			t.Fatalf("add %d should succeed", i+1)
		}
	}

	// 4th should be rejected.
	if ct.TryAdd("10.0.0.1") {
		t.Error("should reject when per-IP limit reached")
	}
	if ct.ActivePerIP("10.0.0.1") != 3 {
		t.Errorf("ActivePerIP = %d, want 3", ct.ActivePerIP("10.0.0.1"))
	}

	// Remove one, then add should succeed again.
	ct.Remove("10.0.0.1")
	if !ct.TryAdd("10.0.0.1") {
		t.Error("should allow after removing one connection")
	}
}

func TestConnTracker_GlobalLimit(t *testing.T) {
	ct := NewConnTracker(ConnTrackerConfig{MaxPerIP: 10, MaxTotal: 5})

	for i := 0; i < 5; i++ {
		ip := fmt.Sprintf("10.0.0.%d", i+1)
		if !ct.TryAdd(ip) {
			t.Fatalf("add for %s should succeed", ip)
		}
	}

	// Global limit reached — new IP should be rejected.
	if ct.TryAdd("10.0.0.99") {
		t.Error("should reject when global limit reached")
	}
	if ct.ActiveTotal() != 5 {
		t.Errorf("ActiveTotal = %d, want 5", ct.ActiveTotal())
	}

	// Remove one, then a new one should succeed.
	ct.Remove("10.0.0.1")
	if !ct.TryAdd("10.0.0.99") {
		t.Error("should allow after removing one connection")
	}
}

func TestConnTracker_MultipleIPsIndependent(t *testing.T) {
	ct := NewConnTracker(ConnTrackerConfig{MaxPerIP: 2, MaxTotal: 100})

	ct.TryAdd("ip-a")
	ct.TryAdd("ip-a")
	// ip-a is at limit.
	if ct.TryAdd("ip-a") {
		t.Error("ip-a should be at limit")
	}

	// ip-b should still be allowed.
	if !ct.TryAdd("ip-b") {
		t.Error("ip-b should be allowed independently")
	}
	if ct.ActivePerIP("ip-a") != 2 {
		t.Errorf("ip-a ActivePerIP = %d, want 2", ct.ActivePerIP("ip-a"))
	}
	if ct.ActivePerIP("ip-b") != 1 {
		t.Errorf("ip-b ActivePerIP = %d, want 1", ct.ActivePerIP("ip-b"))
	}
}

func TestConnTracker_RemoveUnknownIP(t *testing.T) {
	ct := NewConnTracker(ConnTrackerConfig{MaxPerIP: 10, MaxTotal: 100})

	// Should not panic.
	ct.Remove("unknown-ip")
	if ct.ActiveTotal() != 0 {
		t.Errorf("ActiveTotal = %d, want 0", ct.ActiveTotal())
	}
}

func TestConnTracker_RemoveCleansUpMap(t *testing.T) {
	ct := NewConnTracker(ConnTrackerConfig{MaxPerIP: 10, MaxTotal: 100})

	ct.TryAdd("10.0.0.1")
	ct.Remove("10.0.0.1")

	ct.mu.Lock()
	_, exists := ct.perIP["10.0.0.1"]
	ct.mu.Unlock()

	if exists {
		t.Error("IP entry should be removed from map when count reaches zero")
	}
}

func TestConnTracker_ConcurrentAccess(t *testing.T) {
	ct := NewConnTracker(ConnTrackerConfig{MaxPerIP: 1000, MaxTotal: 10000})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ip := fmt.Sprintf("10.0.0.%d", n%10)
			for j := 0; j < 50; j++ {
				ct.TryAdd(ip)
				ct.Remove(ip)
			}
		}(i)
	}
	wg.Wait()

	// After equal adds and removes, total should be 0.
	if ct.ActiveTotal() != 0 {
		t.Errorf("ActiveTotal = %d, want 0 after balanced add/remove", ct.ActiveTotal())
	}
}

func TestConnTracker_GlobalLimitBeforePerIP(t *testing.T) {
	// Global limit is smaller than per-IP limit.
	ct := NewConnTracker(ConnTrackerConfig{MaxPerIP: 10, MaxTotal: 3})

	for i := 0; i < 3; i++ {
		if !ct.TryAdd("10.0.0.1") {
			t.Fatalf("add %d should succeed", i+1)
		}
	}

	// Per-IP limit is 10 but global is 3 — should be rejected.
	if ct.TryAdd("10.0.0.1") {
		t.Error("should reject due to global limit even though per-IP has room")
	}
}
