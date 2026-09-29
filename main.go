package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
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

	if memLimitStr := os.Getenv("GOMEMLIMIT"); memLimitStr != "" {
		if memLimit, parseErr := strconv.ParseInt(memLimitStr, 10, 64); parseErr == nil && memLimit > 0 {
			debug.SetMemoryLimit(memLimit)
			log.Printf("GOMEMLIMIT set to %d bytes (%.1f GiB)", memLimit, float64(memLimit)/(1<<30))
		} else {
			log.Printf("Warning: Invalid GOMEMLIMIT value '%s', ignoring", memLimitStr)
		}
	}

	a, err := app.New(context.Background(), cfg)
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
	log.Fatal(server.ListenAndServe())
}
