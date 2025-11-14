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

	// Create a new server with default configuration
	server := shttp.New(ctx, nil)

	// Use the default middleware stack in one call!
	// This includes: RequestID, UserContext, ContextualLogger, Logging, Recovery
	// All middleware are in optimal order for request processing and logging.
	server.Use(shttp.DefaultMiddlewareStack(logger)...)

	// Register routes
	server.GET("/", homeHandler)
	server.GET("/users/{id}", userHandler)
	server.POST("/api/data", dataHandler)

	// Set up a channel to handle shutdown signals
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	go func() {
		log.Println("Starting server at http://localhost:8080")
		log.Println("Try:")
		log.Println("  curl http://localhost:8080/")
		log.Println("  curl http://localhost:8080/users/123")
		log.Println("  curl -X POST http://localhost:8080/api/data")
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-done
	log.Println("Shutting down server...")
	if err := server.Shutdown(context.Background()); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
}

// homeHandler demonstrates accessing the logger from context with auto-injected request attributes
func homeHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	logger := shttp.GetLogger(ctx)
	if logger == nil {
		return fmt.Errorf("logger not found in context")
	}

	// Log using the context logger - request_id, user_id, client_ip are automatically included!
	logger.Info(ctx, "Handling home request")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintln(w, `<html>
	<head><title>Default Middleware Stack Example</title></head>
	<body>
		<h1>Default Middleware Stack Example</h1>
		<p>This example demonstrates the DefaultMiddlewareStack convenience function.</p>
		<h2>Features</h2>
		<ul>
			<li><strong>RequestID:</strong> Each request gets a unique ID automatically</li>
			<li><strong>UserContext:</strong> User ID is extracted from Authorization header</li>
			<li><strong>ContextualLogger:</strong> Logger is injected with request metadata (request_id, user_id, client_ip)</li>
			<li><strong>Logging:</strong> Request/response details are logged automatically</li>
			<li><strong>Recovery:</strong> Panics are caught and logged</li>
		</ul>
		<h2>Try These URLs</h2>
		<ul>
			<li><a href="/users/alice">GET /users/alice</a></li>
			<li><a href="/users/bob">GET /users/bob</a></li>
			<li>POST /api/data</li>
		</ul>
		<p>Check the server logs to see structured logging with auto-injected request attributes.</p>
	</body>
</html>`)
	return nil
}

// userHandler demonstrates retrieving path parameters and logging with context attributes
func userHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	logger := shttp.GetLogger(ctx)
	if logger == nil {
		return fmt.Errorf("logger not found in context")
	}

	// Get path parameter
	userID := shttp.PathValue(r, "id")

	// Log with context - request_id, user_id, client_ip are already in the log!
	logger.Info(ctx, "Fetching user details", "path_param_id", userID)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"id": "%s", "name": "User %s", "timestamp": "%s"}`, userID, userID, time.Now().Format(time.RFC3339))
	return nil
}

// dataHandler demonstrates a POST handler with logging
func dataHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	logger := shttp.GetLogger(ctx)
	if logger == nil {
		return fmt.Errorf("logger not found in context")
	}

	// Log the data submission
	logger.Info(ctx, "Processing data submission")

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "success", "timestamp": "%s"}`, time.Now().Format(time.RFC3339))
	return nil
}
