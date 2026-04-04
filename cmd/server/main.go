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
)

func main() {
	config := server.Config{
		Host:       getEnv("HOST", "0.0.0.0"),
		Port:       getEnv("PORT", "8080"),
		TrustProxy: strings.EqualFold(getEnv("TRUST_PROXY", "false"), "true"),
	}

	manager := server.NewRoomManager()
	limiter := server.NewRateLimiter(server.DefaultRateLimitConfig())

	stopGC := manager.StartGC()
	defer stopGC()

	srv := server.NewServer(config, manager, limiter)

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

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
