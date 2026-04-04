package server

import (
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
	Host       string
	Port       string
	TrustProxy bool
}

// HealthResponse is returned by the health endpoint.
type HealthResponse struct {
	Status      string `json:"status"`
	Rooms       int    `json:"rooms"`
	Connections int    `json:"connections"`
	Uptime      string `json:"uptime"`
}

// NewServer creates and configures the HTTP server.
func NewServer(config Config, manager *RoomManager, limiter *RateLimiter) *http.Server {
	mux := http.NewServeMux()

	// Health check.
	mux.HandleFunc("/health", handleHealth(manager))

	// WebSocket endpoint.
	mux.HandleFunc("/ws/", HandleWebSocket(manager, limiter, config.TrustProxy))

	// SPA fallback: serve static files or index.html.
	mux.HandleFunc("/", handleSPA())

	addr := fmt.Sprintf("%s:%s", config.Host, config.Port)
	return &http.Server{
		Addr:        addr,
		Handler:     mux,
		IdleTimeout: 60 * time.Second,
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

func handleSPA() http.HandlerFunc {
	// Try to find web/dist directory for static files.
	distDir := findDistDir()

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// If web/dist exists, serve static files with SPA fallback.
		if distDir != "" {
			path := filepath.Join(distDir, filepath.Clean(r.URL.Path))

			// Check if the file exists.
			info, err := os.Stat(path)
			if err == nil && !info.IsDir() {
				http.ServeFile(w, r, path)
				return
			}

			// SPA fallback: serve index.html for non-file routes.
			indexPath := filepath.Join(distDir, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				http.ServeFile(w, r, indexPath)
				return
			}
		}

		// No frontend built yet — serve placeholder.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, placeholderHTML)
	}
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
