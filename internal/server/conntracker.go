package server

import "sync"

// ConnTrackerConfig holds connection limit settings.
type ConnTrackerConfig struct {
	MaxPerIP int
	MaxTotal int
}

// DefaultConnTrackerConfig returns the default connection limit settings.
func DefaultConnTrackerConfig() ConnTrackerConfig {
	return ConnTrackerConfig{
		MaxPerIP: 100,
		MaxTotal: 1000,
	}
}

// ConnTracker tracks active WebSocket connections per IP and globally.
// It enforces concurrent connection limits to prevent resource exhaustion.
type ConnTracker struct {
	mu     sync.Mutex
	perIP  map[string]int
	total  int
	config ConnTrackerConfig
}

// NewConnTracker creates a connection tracker with the given config.
func NewConnTracker(config ConnTrackerConfig) *ConnTracker {
	return &ConnTracker{
		perIP:  make(map[string]int),
		config: config,
	}
}

// TryAdd attempts to register a new connection for the given IP.
// Returns true if the connection is allowed (within both per-IP and global limits).
// If allowed, the counters are atomically incremented.
func (ct *ConnTracker) TryAdd(ip string) bool {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if ct.total >= ct.config.MaxTotal {
		return false
	}
	if ct.perIP[ip] >= ct.config.MaxPerIP {
		return false
	}

	ct.perIP[ip]++
	ct.total++
	return true
}

// Remove decrements the connection counters for the given IP.
// Safe to call even if the IP has no tracked connections.
func (ct *ConnTracker) Remove(ip string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	count, ok := ct.perIP[ip]
	if !ok || count <= 0 {
		return
	}

	ct.perIP[ip]--
	ct.total--

	if ct.perIP[ip] == 0 {
		delete(ct.perIP, ip)
	}
}

// ActivePerIP returns the number of active connections for the given IP.
func (ct *ConnTracker) ActivePerIP(ip string) int {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return ct.perIP[ip]
}

// ActiveTotal returns the total number of active connections.
func (ct *ConnTracker) ActiveTotal() int {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return ct.total
}
