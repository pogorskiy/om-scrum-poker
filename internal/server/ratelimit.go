package server

import (
	"sync"
	"time"
)

// TokenBucket implements a simple token bucket rate limiter.
type TokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

// RateLimiter tracks rate limits per IP address.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*TokenBucket
	config  RateLimitConfig
	stop    chan struct{}
}

// RateLimitConfig holds rate limiter settings.
type RateLimitConfig struct {
	RoomCreationsPerMin int
	WSConnectionsPerMin int
}

// DefaultRateLimitConfig returns the default rate limit settings.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RoomCreationsPerMin: 5,
		WSConnectionsPerMin: 20,
	}
}

// NewRateLimiter creates a rate limiter with the given config.
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*TokenBucket),
		config:  config,
		stop:    make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

// Close stops the cleanup goroutine.
func (rl *RateLimiter) Close() {
	close(rl.stop)
}

// AllowRoomCreation checks if the IP can create a room.
func (rl *RateLimiter) AllowRoomCreation(ip string) bool {
	key := "room:" + ip
	max := float64(rl.config.RoomCreationsPerMin)
	rate := max / 60.0
	return rl.allow(key, max, rate)
}

// AllowWSConnection checks if the IP can open a WebSocket connection.
func (rl *RateLimiter) AllowWSConnection(ip string) bool {
	key := "ws:" + ip
	max := float64(rl.config.WSConnectionsPerMin)
	rate := max / 60.0
	return rl.allow(key, max, rate)
}

func (rl *RateLimiter) allow(key string, maxTokens, refillRate float64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, ok := rl.buckets[key]
	if !ok {
		bucket = &TokenBucket{
			tokens:     maxTokens - 1,
			maxTokens:  maxTokens,
			refillRate: refillRate,
			lastRefill: now,
		}
		rl.buckets[key] = bucket
		return true
	}

	// Refill tokens based on elapsed time.
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * bucket.refillRate
	if bucket.tokens > bucket.maxTokens {
		bucket.tokens = bucket.maxTokens
	}
	bucket.lastRefill = now

	if bucket.tokens < 1 {
		return false
	}

	bucket.tokens--
	return true
}

// cleanup removes stale buckets every 5 minutes.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-rl.stop:
			return
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for key, bucket := range rl.buckets {
				if now.Sub(bucket.lastRefill) > 5*time.Minute {
					delete(rl.buckets, key)
				}
			}
			rl.mu.Unlock()
		}
	}
}
