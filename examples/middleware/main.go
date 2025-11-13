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

	"github.com/andres-vara/shttp"
	"github.com/andres-vara/slogr"
)

func main() {
	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a logger
	logger := slogr.New(os.Stdout, slogr.DefaultOptions())

	// Create server configuration with the logger
	config := &shttp.Config{
		Addr:           ":8080",
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
		Logger:         logger,
	}

	// Create a new server with our configuration
	server := shttp.New(ctx, config)

	// Add middleware to the server
	// 1. Request ID middleware adds a unique ID to each request
	server.Use(shttp.RequestIDMiddleware())

	// 2. Recovery middleware catches panics in handlers
	server.Use(shttp.RecoveryMiddleware(logger))

	// 3. Logging middleware logs request details
	server.Use(shttp.LoggingMiddleware(logger))

	// 4. CORS middleware for cross-origin requests
	server.Use(shttp.CORSMiddleware([]string{"*"}))

	// 5. Timeout middleware sets a timeout for request processing
	server.Use(shttp.TimeoutMiddleware(5 * time.Second))

	// 6. Custom middleware
	server.Use(customHeaderMiddleware("X-Server", "shttp-example"))

	// Register routes
	server.GET("/", helloWorldHandler)
	server.GET("/slow", slowHandler)
	server.GET("/panic", panicHandler)

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

	// Attempt graceful shutdown using the timeout context
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server gracefully stopped")
}

// helloWorldHandler returns a simple hello world message
func helloWorldHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	requestID := shttp.GetRequestID(ctx)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Hello, World! (Request ID: %s)\n", requestID)
	return nil
}

// slowHandler simulates a slow response
func slowHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(3 * time.Second):
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "This was a slow response")
		return nil
	}
}

// panicHandler deliberately causes a panic
func panicHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	panic("This is a deliberate panic that will be caught by the recovery middleware")
}

// customHeaderMiddleware is a custom middleware that adds a header to each response
func customHeaderMiddleware(headerName, headerValue string) shttp.Middleware {
	return func(next shttp.Handler) shttp.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			w.Header().Set(headerName, headerValue)
			return next(ctx, w, r)
		}
	}
}
