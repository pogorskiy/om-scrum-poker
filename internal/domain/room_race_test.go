package domain

import (
	"sync"
	"testing"
)

func TestLastActivity_ConcurrentAccess(t *testing.T) {
	r, _ := NewRoom("room1", "Test")

	var wg sync.WaitGroup
	const goroutines = 50

	// Half write, half read.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		if i%2 == 0 {
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					r.TouchActivity()
				}
			}()
		} else {
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					_ = r.GetLastActivity()
				}
			}()
		}
	}

	wg.Wait()

	// Sanity check: activity should be set.
	if r.LastActivityUnixNano() <= 0 {
		t.Error("expected positive LastActivityUnixNano after concurrent access")
	}
}
