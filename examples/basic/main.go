package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/andres-vara/shttp"
	"github.com/andres-vara/slogr"
)

func main() {
	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slogr.New(os.Stdout, slogr.DefaultOptions())

	// Create a new server with default configuration
	server := shttp.New(ctx, nil)

	server.Use(shttp.LoggerMiddleware(logger))
	server.Use(shttp.LoggingMiddleware(logger))

	// Register routes
	server.GET("/", helloWorldHandler)
	server.GET("/health", healthCheckHandler)
	server.GET("/users/{id}", userHandler)
	server.GET("/test/{param1}", testHandler)

	// Set up a channel to handle shutdown signals
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	go func() {
		log.Println("Starting server at http://localhost:8080")
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-done
	log.Println("Server is shutting down...")

	// Create a deadline for the shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server gracefully stopped")
}

// helloWorldHandler returns a simple hello world message
func helloWorldHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	logger := shttp.GetLogger(ctx)
	logger.Infof(ctx, "Handling hello world request path: %v", r.URL.Path)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Hello, World!")
	return nil
}

// healthCheckHandler returns a 200 OK response for health checks
func healthCheckHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"status": "healthy"}`)
	return nil
}

// userHandler demonstrates accessing path parameters from the URL
func userHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	userID := shttp.PathValue(r, "id")

	// Debug information
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Path: %s\n", r.URL.Path)
	fmt.Fprintf(w, "UserID from PathValue: %s\n", userID)

	// Manual path extraction as fallback
	parts := strings.Split(r.URL.Path, "/")
	manualID := ""
	if len(parts) > 2 {
		manualID = parts[len(parts)-1]
	}
	fmt.Fprintf(w, "Manually extracted ID: %s\n", manualID)

	return nil
}

// testHandler is a test route for debugging path parameters
func testHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	param1 := shttp.PathValue(r, "param1")

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	// param2 := r.PathValue("param2")

	fmt.Fprintf(w, "Path: %s\n", r.URL.Path)
	fmt.Fprintf(w, "param1: %s\n", param1)
	// fmt.Fprintf(w, "param2: %s\n", param2)

	return nil
}
