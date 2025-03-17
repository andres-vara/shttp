package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/andres-vara/slogr"
)

// Context key
type contextKey string

const (
	loggerKey contextKey = "logger"
)

// GetLogger retrieves the logger from the context
func GetLogger(ctx context.Context) *slogr.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slogr.Logger); ok {
		return logger
	}
	return nil
}

func main() {
	// Create a logger
	logger := slogr.New(os.Stdout, slogr.DefaultOptions())
	logger.Info(context.Background(), "Starting direct logger example (without router)")

	// Create a simple handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Try to get logger from context
		ctx := r.Context()
		loggerFromCtx := GetLogger(ctx)
		
		if loggerFromCtx == nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "Logger not found in context")
			return
		}
		
		// Logger was found!
		loggerFromCtx.Info(ctx, "Handler called with logger from context")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Success! Logger found in context")
	})

	// Wrap our handler in middleware
	originalHandler := http.DefaultServeMux
	
	// Create middleware handler that adds logger to context
	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add logger to context
		ctx := context.WithValue(r.Context(), loggerKey, logger)
		
		// Call the next handler with updated context
		originalHandler.ServeHTTP(w, r.WithContext(ctx))
	})

	// Log startup
	logger.Info(context.Background(), "Server starting on :8080")
	
	// Start server
	if err := http.ListenAndServe(":8080", wrappedHandler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
} 