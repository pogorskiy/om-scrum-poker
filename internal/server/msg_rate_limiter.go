package server

import (
	"sync"
	"time"
)

// MsgRateLimiter is a per-connection token bucket rate limiter for WebSocket messages.
type MsgRateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

// NewMsgRateLimiter creates a message rate limiter with the given capacity and refill rate.
// capacity is the max burst size, refillRate is tokens added per second.
func NewMsgRateLimiter(capacity float64, refillRate float64) *MsgRateLimiter {
	return &MsgRateLimiter{
		tokens:     capacity,
		maxTokens:  capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// DefaultMsgRateLimiter creates a rate limiter allowing 20 msg/sec with burst of 20.
func DefaultMsgRateLimiter() *MsgRateLimiter {
	return NewMsgRateLimiter(20, 20)
}

// Allow checks if a message is allowed. Returns true if the message can proceed.
func (m *MsgRateLimiter) Allow() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(m.lastRefill).Seconds()
	m.tokens += elapsed * m.refillRate
	if m.tokens > m.maxTokens {
		m.tokens = m.maxTokens
	}
	m.lastRefill = now

	if m.tokens < 1 {
		return false
	}

	m.tokens--
	return true
}
