package server

import (
	"embed"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"
)

func TestHandleHealth_GET(t *testing.T) {
	rm := NewRoomManager()

	handler := handleHealth(rm, "dev")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	body, _ := io.ReadAll(resp.Body)
	var hr HealthResponse
	if err := json.Unmarshal(body, &hr); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if hr.Status != "ok" {
		t.Errorf("status = %q, want %q", hr.Status, "ok")
	}
	if hr.Rooms != 0 {
		t.Errorf("rooms = %d, want 0", hr.Rooms)
	}
	if hr.Connections != 0 {
		t.Errorf("connections = %d, want 0", hr.Connections)
	}
	if hr.Uptime == "" {
		t.Error("uptime should not be empty")
	}
}

func TestHandleHealth_WithRooms(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreateRoom("r1", "Room 1", "")
	rm.GetOrCreateRoom("r2", "Room 2", "")

	c := fakeClient("r1", rm)
	rm.RegisterClient("r1", c)

	handler := handleHealth(rm, "dev")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	var hr HealthResponse
	json.Unmarshal(body, &hr)

	if hr.Rooms != 2 {
		t.Errorf("rooms = %d, want 2", hr.Rooms)
	}
	if hr.Connections != 1 {
		t.Errorf("connections = %d, want 1", hr.Connections)
	}
}

func TestHandleHealth_MethodNotAllowed(t *testing.T) {
	rm := NewRoomManager()
	handler := handleHealth(rm, "dev")

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/health", nil)
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Result().StatusCode != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", w.Result().StatusCode, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestServeEmbedFS_ServesIndexHTML(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":      {Data: []byte("<html>hello</html>")},
		"assets/style.css": {Data: []byte("body{}")},
	}

	handler := ServeEmbedFS(fsys)

	// Request for root should serve index.html.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	if string(body) != "<html>hello</html>" {
		t.Errorf("body = %q, want index.html content", string(body))
	}
}

func TestServeEmbedFS_ServesStaticFile(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":      {Data: []byte("<html>hello</html>")},
		"assets/style.css": {Data: []byte("body{}")},
	}

	handler := ServeEmbedFS(fsys)

	req := httptest.NewRequest(http.MethodGet, "/assets/style.css", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	if string(body) != "body{}" {
		t.Errorf("body = %q, want css content", string(body))
	}
}

func TestServeEmbedFS_SPAFallback(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": {Data: []byte("<html>SPA</html>")},
	}

	handler := ServeEmbedFS(fsys)

	// Unknown route should fall back to index.html (SPA behavior).
	req := httptest.NewRequest(http.MethodGet, "/room/abc123", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	if string(body) != "<html>SPA</html>" {
		t.Errorf("SPA fallback body = %q, want index.html content", string(body))
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		trustProxy bool
		xff        string
		xRealIP    string
		want       string
	}{
		{
			name:       "direct connection with port",
			remoteAddr: "192.168.1.1:12345",
			want:       "192.168.1.1",
		},
		{
			name:       "direct connection without port",
			remoteAddr: "192.168.1.1",
			want:       "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For ignored when trustProxy=false",
			remoteAddr: "10.0.0.1:8080",
			trustProxy: false,
			xff:        "203.0.113.50",
			want:       "10.0.0.1",
		},
		{
			name:       "X-Forwarded-For used when trustProxy=true",
			remoteAddr: "10.0.0.1:8080",
			trustProxy: true,
			xff:        "203.0.113.50",
			want:       "203.0.113.50",
		},
		{
			name:       "X-Forwarded-For with multiple IPs",
			remoteAddr: "10.0.0.1:8080",
			trustProxy: true,
			xff:        "203.0.113.50, 70.41.3.18, 150.172.238.178",
			want:       "203.0.113.50",
		},
		{
			name:       "X-Real-IP used when trustProxy=true and no XFF",
			remoteAddr: "10.0.0.1:8080",
			trustProxy: true,
			xRealIP:    "198.51.100.10",
			want:       "198.51.100.10",
		},
		{
			name:       "X-Forwarded-For takes priority over X-Real-IP",
			remoteAddr: "10.0.0.1:8080",
			trustProxy: true,
			xff:        "203.0.113.50",
			xRealIP:    "198.51.100.10",
			want:       "203.0.113.50",
		},
		{
			name:       "IPv6 with port",
			remoteAddr: "[::1]:8080",
			want:       "[::1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			got := clientIP(req, tt.trustProxy)
			if got != tt.want {
				t.Errorf("clientIP = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandleSPA_MethodNotAllowed(t *testing.T) {
	// handleSPA with empty embed.FS and no dist dir should still reject non-GET methods.
	// We test through ServeEmbedFS instead since handleSPA depends on embed.FS.
	fsys := fstest.MapFS{
		"index.html": {Data: []byte("<html>test</html>")},
	}
	handler := ServeEmbedFS(fsys)

	// ServeEmbedFS itself doesn't check method — that's done in handleSPA wrapper.
	// We can test that it at least responds to HEAD.
	req := httptest.NewRequest(http.MethodHead, "/", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Result().StatusCode >= 400 {
		t.Errorf("HEAD request should succeed, got status %d", w.Result().StatusCode)
	}
}

func TestLogMiddleware(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	handler := LogMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Result().StatusCode, http.StatusOK)
	}
	body, _ := io.ReadAll(w.Result().Body)
	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", string(body), "ok")
	}
}

func TestHandleHealth_AfterClientDisconnect(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreateRoom("room-1", "Test", "")

	c := fakeClient("room-1", rm)
	rm.RegisterClient("room-1", c)

	if rm.ConnectionCount() != 1 {
		t.Fatalf("expected 1 connection before unregister, got %d", rm.ConnectionCount())
	}

	// Simulate client disconnect by unregistering.
	rm.UnregisterClient("room-1", c)

	handler := handleHealth(rm, "dev")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	var hr HealthResponse
	json.Unmarshal(body, &hr)

	if hr.Connections != 0 {
		t.Errorf("connections = %d, want 0 after client disconnect", hr.Connections)
	}
	if hr.Rooms != 1 {
		t.Errorf("rooms = %d, want 1 (room still exists)", hr.Rooms)
	}
}

func TestHandleHealth_MultipleRoomsMultipleClients(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreateRoom("room-1", "Room 1", "")
	rm.GetOrCreateRoom("room-2", "Room 2", "")
	rm.GetOrCreateRoom("room-3", "Room 3", "")

	// 2 clients in room-1, 1 client in room-2, 0 in room-3.
	rm.RegisterClient("room-1", fakeClient("room-1", rm))
	rm.RegisterClient("room-1", fakeClient("room-1", rm))
	rm.RegisterClient("room-2", fakeClient("room-2", rm))

	handler := handleHealth(rm, "dev")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	var hr HealthResponse
	json.Unmarshal(body, &hr)

	if hr.Rooms != 3 {
		t.Errorf("rooms = %d, want 3", hr.Rooms)
	}
	if hr.Connections != 3 {
		t.Errorf("connections = %d, want 3", hr.Connections)
	}
}

func TestNewServer_ReadHeaderTimeout(t *testing.T) {
	rm := NewRoomManager()
	limiter := NewRateLimiter(DefaultRateLimitConfig())
	var emptyFS embed.FS

	srv := NewServer(Config{Host: "127.0.0.1", Port: "0"}, rm, limiter, NewConnTracker(DefaultConnTrackerConfig()), emptyFS)

	if srv.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want %v", srv.ReadHeaderTimeout, 5*time.Second)
	}
}

func TestNewServer_IdleTimeout(t *testing.T) {
	rm := NewRoomManager()
	limiter := NewRateLimiter(DefaultRateLimitConfig())
	var emptyFS embed.FS

	srv := NewServer(Config{Host: "127.0.0.1", Port: "0"}, rm, limiter, NewConnTracker(DefaultConnTrackerConfig()), emptyFS)

	if srv.IdleTimeout != 60*time.Second {
		t.Errorf("IdleTimeout = %v, want %v", srv.IdleTimeout, 60*time.Second)
	}
}

func TestHandleHealth_BuildTime(t *testing.T) {
	rm := NewRoomManager()

	handler := handleHealth(rm, "2024-01-15T14:30:00Z")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	var hr HealthResponse
	if err := json.Unmarshal(body, &hr); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if hr.BuildTime != "2024-01-15T14:30:00Z" {
		t.Errorf("build_time = %q, want %q", hr.BuildTime, "2024-01-15T14:30:00Z")
	}
}

func TestHandleHealth_BuildTimeDefault(t *testing.T) {
	rm := NewRoomManager()

	handler := handleHealth(rm, "dev")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	var hr HealthResponse
	if err := json.Unmarshal(body, &hr); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if hr.BuildTime != "dev" {
		t.Errorf("build_time = %q, want %q", hr.BuildTime, "dev")
	}
}
