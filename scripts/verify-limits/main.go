// verify-limits checks that rate limiting and connection limits work on an
// om-scrum-poker server. Each test runs independently with its own cooldown
// period so token buckets refill between tests.
//
// Usage:
//
//	go run ./scripts/verify-limits -url wss://example.com
//	go run ./scripts/verify-limits -url ws://localhost:8080
//	go run ./scripts/verify-limits -url wss://example.com -test concurrent
//	go run ./scripts/verify-limits -url wss://example.com -test rate
//	go run ./scripts/verify-limits -url wss://example.com -test rooms
//	go run ./scripts/verify-limits -url wss://example.com -test all
//
// Available tests:
//
//	concurrent  — per-IP concurrent connection limit (default 100)
//	rate        — WebSocket connection rate limit (default 20/min)
//	rooms       — room creation rate limit (default 5/min)
//	all         — run all tests sequentially with cooldowns (default)
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

func main() {
	urlFlag := flag.String("url", "", "WebSocket URL (e.g. wss://example.com or ws://localhost:8080)")
	testFlag := flag.String("test", "all", "Test to run: concurrent, rate, rooms, or all")
	cooldownFlag := flag.Duration("cooldown", 65*time.Second, "Cooldown between tests to let token buckets refill")
	flag.Parse()

	if *urlFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: -url is required")
		fmt.Fprintln(os.Stderr, "Usage: go run ./scripts/verify-limits -url wss://example.com [-test all]")
		os.Exit(2)
	}

	cfg := parseTarget(*urlFlag)

	fmt.Println("=== om-scrum-poker Limit Verification ===")
	fmt.Printf("Target:   %s\n", cfg.wsURL)
	fmt.Printf("Origin:   %s\n", cfg.origin)
	fmt.Printf("Test:     %s\n", *testFlag)
	fmt.Printf("Cooldown: %s\n\n", *cooldownFlag)

	if !checkHealth(cfg) {
		fmt.Println("FAIL: Server not reachable. Aborting.")
		os.Exit(1)
	}
	fmt.Println()

	tests := selectTests(*testFlag)
	if tests == nil {
		fmt.Fprintf(os.Stderr, "Unknown test: %q (use concurrent, rate, rooms, or all)\n", *testFlag)
		os.Exit(2)
	}

	passed, total := 0, len(tests)
	for i, t := range tests {
		if t.run(cfg) {
			passed++
		}
		// Cooldown between tests so token buckets refill.
		if i < len(tests)-1 {
			fmt.Printf("\n  Waiting %s for rate limit buckets to refill...\n\n", *cooldownFlag)
			time.Sleep(*cooldownFlag)
		}
	}

	fmt.Printf("\n=== Results: %d/%d tests passed ===\n", passed, total)
	if passed < total {
		os.Exit(1)
	}
}

// --- config ---

type config struct {
	wsURL  string // wss://example.com or ws://localhost:8080
	origin string // https://example.com or http://localhost:8080
}

func parseTarget(raw string) config {
	var c config
	c.wsURL = raw
	if strings.HasPrefix(raw, "wss://") {
		c.origin = "https://" + strings.TrimPrefix(raw, "wss://")
	} else if strings.HasPrefix(raw, "ws://") {
		c.origin = "http://" + strings.TrimPrefix(raw, "ws://")
	} else {
		fmt.Fprintf(os.Stderr, "Error: -url must start with ws:// or wss://\n")
		os.Exit(2)
	}
	return c
}

// --- test registry ---

type testCase struct {
	name string
	run  func(config) bool
}

func selectTests(name string) []testCase {
	all := []testCase{
		{"concurrent", testConcurrentConnections},
		{"rate", testConnectionRateLimit},
		{"rooms", testRoomCreationRateLimit},
	}
	if name == "all" {
		return all
	}
	for _, t := range all {
		if t.name == name {
			return []testCase{t}
		}
	}
	return nil
}

// --- health check ---

func checkHealth(cfg config) bool {
	url := cfg.origin + "/health"
	fmt.Printf("Health check: %s ... ", url)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("ERROR: HTTP %d\n", resp.StatusCode)
		return false
	}
	fmt.Println("OK")
	return true
}

// --- Test 1: concurrent connections ---

func testConcurrentConnections(cfg config) bool {
	fmt.Println("[concurrent] Per-IP concurrent connection limit")
	fmt.Println("  Strategy: open connections one-by-one with pauses to stay under")
	fmt.Println("  rate limit (20/min). Hold them all open. Send keepalive pings")
	fmt.Println("  to prevent idle timeouts. Expect rejection at ~100.")
	fmt.Println()

	roomID := "verify-conns-" + randomHex(4)
	type liveConn struct {
		conn   *websocket.Conn
		ctx    context.Context
		cancel context.CancelFunc
	}
	var conns []liveConn
	var mu sync.Mutex

	ctx := context.Background()
	maxAttempts := 115 // slightly above expected 100 limit

	// Token bucket refills at 20/60 = 0.33 tokens/sec.
	// Open 1 connection every 3.5 seconds = ~0.29/sec, always under refill.
	delay := 3500 * time.Millisecond

	fmt.Printf("  Opening connections (1 every %s to avoid rate limit)...\n", delay)
	fmt.Println("  This will take ~6-7 minutes for 100+ connections.")
	fmt.Println()

	totalAccepted, totalRejected := 0, 0
	consecutiveRejects := 0

	for i := 0; i < maxAttempts; i++ {
		conn, err := dialWS(ctx, cfg, roomID)
		if err != nil {
			totalRejected++
			consecutiveRejects++
			if totalRejected == 1 {
				fmt.Printf("  >>> First rejection at attempt #%d (after %d accepted)\n", i+1, totalAccepted)
			}
			if consecutiveRejects >= 5 {
				fmt.Printf("  Stopping after %d consecutive rejections\n", consecutiveRejects)
				break
			}
		} else {
			totalAccepted++
			consecutiveRejects = 0

			// Start a keepalive goroutine — sends a ping every 20s to prevent
			// Caddy/server from considering the connection idle.
			connCtx, connCancel := context.WithCancel(ctx)
			lc := liveConn{conn: conn, ctx: connCtx, cancel: connCancel}

			mu.Lock()
			conns = append(conns, lc)
			mu.Unlock()

			go func(c *websocket.Conn, ctx context.Context) {
				ticker := time.NewTicker(20 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						c.Ping(ctx)
					}
				}
			}(conn, connCtx)
		}

		if (i+1)%10 == 0 || totalRejected > 0 && (i+1)%5 == 0 {
			fmt.Printf("  Progress: %d accepted, %d rejected, %d attempts\n",
				totalAccepted, totalRejected, i+1)
		}

		if i < maxAttempts-1 {
			time.Sleep(delay)
		}
	}

	fmt.Printf("\n  Cleaning up %d connections...\n", len(conns))
	for _, lc := range conns {
		lc.cancel()
		lc.conn.Close(websocket.StatusNormalClosure, "done")
	}

	fmt.Printf("  Total: %d accepted, %d rejected out of %d attempts\n",
		totalAccepted, totalRejected, totalAccepted+totalRejected)

	if totalRejected > 0 && totalAccepted >= 50 {
		fmt.Printf("  PASS: concurrent limit enforced at ~%d connections\n", totalAccepted)
		return true
	}
	if totalRejected == 0 {
		fmt.Printf("  WARN: all %d accepted — limit may be higher than %d\n", totalAccepted, maxAttempts)
		return false
	}
	if totalRejected > 0 && totalAccepted >= 20 {
		fmt.Printf("  PASS: limit enforced at %d connections\n", totalAccepted)
		return true
	}
	fmt.Println("  FAIL: unexpected pattern")
	return false
}

// --- Test 2: connection rate limit ---

func testConnectionRateLimit(cfg config) bool {
	fmt.Println("[rate] WebSocket connection rate limit (expecting ~20/min)")
	fmt.Println("  Strategy: burst 50 rapid connections (open+close). The token bucket")
	fmt.Println("  holds max 20 tokens, so we expect ~20 accepted and ~30 rejected.")
	fmt.Println()

	roomID := "verify-rate-" + randomHex(4)
	ctx := context.Background()
	accepted, rejected := 0, 0
	attempts := 50

	for i := 0; i < attempts; i++ {
		conn, err := dialWS(ctx, cfg, roomID)
		if err != nil {
			rejected++
			if rejected == 1 {
				fmt.Printf("  First rejection at attempt #%d (after %d accepted)\n", i+1, accepted)
			}
		} else {
			accepted++
			conn.Close(websocket.StatusNormalClosure, "done")
		}
		// Tiny delay to keep ordering predictable, but fast enough to outpace refill.
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Printf("\n  Total: %d accepted, %d rejected out of %d\n", accepted, rejected, attempts)

	if rejected > 0 && accepted >= 10 {
		fmt.Printf("  PASS: rate limit kicked in after %d connections\n", accepted)
		return true
	}
	if rejected == 0 {
		fmt.Printf("  WARN: all %d accepted — rate limit may not be working\n", attempts)
		return false
	}
	if rejected > 0 && accepted > 0 {
		fmt.Printf("  PASS: rate limit active (accepted %d, rejected %d)\n", accepted, rejected)
		return true
	}
	fmt.Println("  FAIL: all rejected — bucket may already be empty from previous test")
	return false
}

// --- Test 3: room creation rate limit ---

func testRoomCreationRateLimit(cfg config) bool {
	fmt.Println("[rooms] Room creation rate limit (expecting ~5/min)")
	fmt.Println("  Strategy: connect to unique room IDs and send join to create rooms.")
	fmt.Println()

	ctx := context.Background()
	sessionID := randomHex(16)
	accepted, rejected := 0, 0
	attempts := 10

	for i := 0; i < attempts; i++ {
		roomID := fmt.Sprintf("verify-room-%s-%d", randomHex(4), i)

		conn, err := dialWS(ctx, cfg, roomID)
		if err != nil {
			rejected++
			fmt.Printf("  Room #%2d: connection rejected (429)\n", i+1)
			continue
		}

		// Send join — this triggers room creation on the server.
		join := map[string]interface{}{
			"type": "join",
			"payload": map[string]string{
				"sessionId": sessionID,
				"userName":  "verify-bot",
			},
		}
		data, _ := json.Marshal(join)
		if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
			conn.Close(websocket.StatusNormalClosure, "")
			rejected++
			fmt.Printf("  Room #%2d: write failed\n", i+1)
			continue
		}

		// Read response and check for rate_limited error.
		readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, resp, err := conn.Read(readCtx)
		cancel()
		conn.Close(websocket.StatusNormalClosure, "done")

		if err != nil {
			rejected++
			fmt.Printf("  Room #%2d: read failed\n", i+1)
			continue
		}

		var env struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(resp, &env); err != nil {
			rejected++
			continue
		}

		if env.Type == "error" {
			var ep struct {
				Code string `json:"code"`
			}
			json.Unmarshal(env.Payload, &ep)
			rejected++
			if ep.Code == "rate_limited" {
				fmt.Printf("  Room #%2d: rate_limited\n", i+1)
			} else {
				fmt.Printf("  Room #%2d: error %s\n", i+1, ep.Code)
			}
			continue
		}

		accepted++
		fmt.Printf("  Room #%2d: created OK (%s)\n", i+1, env.Type)
	}

	fmt.Printf("\n  Total: %d created, %d rejected out of %d\n", accepted, rejected, attempts)

	if rejected > 0 && accepted >= 2 {
		fmt.Printf("  PASS: room creation limit kicked in after %d rooms\n", accepted)
		return true
	}
	if rejected == 0 {
		fmt.Printf("  WARN: all %d created — limit may not be working\n", attempts)
		return false
	}
	if rejected > 0 && accepted > 0 {
		fmt.Printf("  PASS: room creation limit active (%d created, %d rejected)\n", accepted, rejected)
		return true
	}
	fmt.Println("  FAIL: all rejected — server may be unreachable or rate limit already spent")
	return false
}

// --- helpers ---

func dialWS(ctx context.Context, cfg config, roomID string) (*websocket.Conn, error) {
	url := cfg.wsURL + "/ws/" + roomID
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(dialCtx, url, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Origin": []string{cfg.origin},
		},
	})
	return conn, err
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
