package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config holds the server configuration.
type Config struct {
	Host           string
	Port           string
	TrustProxy     bool
	AllowedOrigins []string
}

// HealthResponse is returned by the health endpoint.
type HealthResponse struct {
	Status      string `json:"status"`
	Rooms       int    `json:"rooms"`
	Connections int    `json:"connections"`
	Uptime      string `json:"uptime"`
}

// NewServer creates and configures the HTTP server.
// The embedFS parameter should be the embedded web/dist filesystem; it may
// be empty (zero value) for development when the frontend is served by Vite.
func NewServer(config Config, manager *RoomManager, limiter *RateLimiter, embedFS embed.FS) *http.Server {
	mux := http.NewServeMux()

	// Health check.
	mux.HandleFunc("/health", handleHealth(manager))

	// WebSocket endpoint.
	mux.HandleFunc("/ws/", HandleWebSocket(manager, limiter, config.TrustProxy, config.AllowedOrigins))

	// SPA fallback: serve static files or index.html.
	mux.HandleFunc("/", handleSPA(embedFS))

	addr := fmt.Sprintf("%s:%s", config.Host, config.Port)
	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func handleHealth(manager *RoomManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		resp := HealthResponse{
			Status:      "ok",
			Rooms:       manager.RoomCount(),
			Connections: manager.ConnectionCount(),
			Uptime:      manager.Uptime().Round(time.Second).String(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func handleSPA(embedFS embed.FS) http.HandlerFunc {
	// Try embedded FS first (production build baked into the binary).
	embeddedSPA := buildEmbedHandler(embedFS)

	// Fall back to disk-based web/dist (development mode).
	distDir := findDistDir()

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Priority 1: embedded filesystem (self-contained binary).
		if embeddedSPA != nil {
			embeddedSPA.ServeHTTP(w, r)
			return
		}

		// Priority 2: disk-based web/dist (dev server with go run).
		if distDir != "" {
			path := filepath.Join(distDir, filepath.Clean(r.URL.Path))

			info, err := os.Stat(path)
			if err == nil && !info.IsDir() {
				http.ServeFile(w, r, path)
				return
			}

			indexPath := filepath.Join(distDir, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				http.ServeFile(w, r, indexPath)
				return
			}
		}

		// Priority 3: placeholder when no frontend is available.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, placeholderHTML)
	}
}

// buildEmbedHandler returns a SPA handler backed by the embedded FS,
// or nil if the embedded FS contains no real content (only .gitkeep).
func buildEmbedHandler(embedFS embed.FS) http.Handler {
	// The embed is rooted at "dist" inside the web package FS.
	sub, err := fs.Sub(embedFS, "dist")
	if err != nil {
		return nil
	}

	// Check if index.html exists in the embedded FS — if not, the embed
	// only contains .gitkeep and we should fall through to other sources.
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil
	}

	log.Println("Serving frontend from embedded filesystem")
	return ServeEmbedFS(sub)
}

// findDistDir looks for web/dist directory relative to the working directory.
func findDistDir() string {
	candidates := []string{
		"web/dist",
	}

	for _, dir := range candidates {
		info, err := os.Stat(dir)
		if err == nil && info.IsDir() {
			return dir
		}
	}

	return ""
}

// ServeEmbedFS creates a handler from an embedded filesystem (for future use).
func ServeEmbedFS(fsys fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(fsys))

	return func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly.
		path := strings.TrimPrefix(r.URL.Path, "/")
		f, err := fsys.Open(path)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html.
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	}
}

const placeholderHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>OM Scrum Poker</title>
    <style>
        body { font-family: system-ui, sans-serif; display: flex; justify-content: center;
               align-items: center; min-height: 100vh; margin: 0; background: #f5f5f5; }
        .container { text-align: center; padding: 2rem; }
        h1 { color: #333; }
        p { color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h1>OM Scrum Poker</h1>
        <p>Frontend not built yet. The API and WebSocket endpoints are active.</p>
        <p>GET <a href="/health">/health</a> | WS /ws/{roomId}</p>
    </div>
</body>
</html>`

// LogMiddleware is a simple request logging middleware (unused but available).
func LogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
