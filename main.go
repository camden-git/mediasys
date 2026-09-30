package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/camden-git/mediasysbackend/app"
	"github.com/camden-git/mediasysbackend/config"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Info: No .env file found or error loading: %v", err)
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("FATAL: Failed to load configuration: %v", err)
	}

	// GOMEMLIMIT is read by the Go runtime itself (bytes, or units like 2GiB)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	a, err := app.New(ctx, cfg)
	if err != nil {
		log.Fatalf("FATAL: %v", err)
	}
	defer a.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	serverAddr := ":" + port
	fmt.Printf("Server starting on http://localhost:%s\n", port)
	log.Printf("Server listening on %s", serverAddr)
	// no read/write timeouts: uploads and archive downloads can be large and slow
	server := &http.Server{
		Addr:              serverAddr,
		Handler:           a.Router,
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- server.ListenAndServe() }()

	select {
	case err := <-serveErr:
		// ListenAndServe only returns on failure here; deferred Close still runs
		log.Printf("FATAL: server stopped: %v", err)
		return
	case <-ctx.Done():
	}

	log.Println("Shutting down...")
	stop() // a second signal now kills the process immediately
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Warning: graceful shutdown incomplete: %v", err)
		_ = server.Close()
	}
}
