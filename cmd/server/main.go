package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"om-scrum-poker/internal/server"
	"om-scrum-poker/web"
)

func main() {
	config := server.Config{
		Host:           getEnv("HOST", "0.0.0.0"),
		Port:           getEnv("PORT", "8080"),
		TrustProxy:     strings.EqualFold(getEnv("TRUST_PROXY", "false"), "true"),
		AllowedOrigins: parseAllowedOrigins(getEnv("ALLOWED_ORIGINS", "")),
	}

	manager := server.NewRoomManager()
	limiter := server.NewRateLimiter(server.DefaultRateLimitConfig())

	stopGC := manager.StartGC()
	defer stopGC()
	defer limiter.Close()

	srv := server.NewServer(config, manager, limiter, web.DistFS)

	// Graceful shutdown.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Starting OM Scrum Poker on %s:%s", config.Host, config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-shutdown
	log.Println("Shutting down...")

	// Close all WebSocket connections.
	manager.CloseAll()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

// parseAllowedOrigins splits a comma-separated string into origin patterns,
// trimming whitespace and filtering empty entries.
func parseAllowedOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var origins []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
