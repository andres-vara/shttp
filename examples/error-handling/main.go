package main

import (
	"context"
	"errors"
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

// Custom error types for different HTTP status codes
type NotFoundError struct {
	Resource string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("Resource not found: %s", e.Resource)
}

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("Validation error for field %s: %s", e.Field, e.Message)
}

type UnauthorizedError struct {
	Message string
}

func (e UnauthorizedError) Error() string {
	return e.Message
}

func main() {
	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a logger
	logger := slogr.New(os.Stdout, slogr.DefaultOptions())

	// Create a new server with default configuration
	config := &shttp.Config{
		Addr:   ":8080",
		Logger: logger,
	}
	server := shttp.New(ctx, config)

	// Add error handling middleware
	server.Use(errorHandlingMiddleware)

	// Register routes that demonstrate different error types
	server.GET("/success", successHandler)
	server.GET("/not-found", notFoundHandler)
	server.GET("/validation-error", validationErrorHandler)
	server.GET("/unauthorized", unauthorizedHandler)
	server.GET("/server-error", serverErrorHandler)

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

// errorHandlingMiddleware handles different types of errors and maps them to HTTP status codes
func errorHandlingMiddleware(next shttp.Handler) shttp.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		err := next(ctx, w, r)
		if err == nil {
			return nil
		}

		// Set content type for error responses
		w.Header().Set("Content-Type", "application/json")

		// Handle different error types
		switch e := err.(type) {
		case NotFoundError:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error": "not_found", "message": "%s"}`, e.Error())
		case ValidationError:
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error": "validation_error", "field": "%s", "message": "%s"}`, e.Field, e.Message)
		case UnauthorizedError:
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, `{"error": "unauthorized", "message": "%s"}`, e.Error())
		default:
			// Generic server error
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error": "server_error", "message": "%s"}`, err.Error())
		}

		// The error has been handled
		return nil
	}
}

// Handlers that demonstrate different error types

func successHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"status": "success", "message": "Everything is working correctly"}`)
	return nil
}

func notFoundHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	return NotFoundError{Resource: "user"}
}

func validationErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	return ValidationError{Field: "email", Message: "Invalid email format"}
}

func unauthorizedHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	return UnauthorizedError{Message: "Authentication token is invalid or expired"}
}

func serverErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	return errors.New("unexpected internal server error occurred")
} 