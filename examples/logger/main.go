package main

import (
	"context"
	"fmt"
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
	logger.Info(ctx, "Starting logger example server")

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

	// Add middleware to the server - order matters!
	// Logger middleware must come first so it's available in context for other middleware
	server.Use(shttp.LoggerMiddleware(logger))
	server.Use(shttp.RequestIDMiddleware())
	server.Use(shttp.UserContextMiddleware())
	server.Use(shttp.LoggingMiddleware(logger))
	server.Use(shttp.RecoveryMiddleware(logger))
	server.Use(shttp.TimeoutMiddleware(5 * time.Second))

	// Register routes
	server.GET("/", homeHandler)
	server.GET("/debug", debugLogHandler)
	server.GET("/info", infoLogHandler)
	server.GET("/error", errorLogHandler)

	// Set up a channel to handle shutdown signals
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	go func() {
		logger.Info(ctx, "Server is running at http://localhost:8080")
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			logger.Errorf(ctx, "Server error: %v", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-done
	logger.Info(ctx, "Server is shutting down...")

	// Create a deadline for the shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Errorf(ctx, "Server shutdown failed: %v", err)
		os.Exit(1)
	}

	logger.Info(ctx, "Server gracefully stopped")
}

// homeHandler demonstrates basic logger usage
func homeHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// Get the logger from context
	logger := shttp.GetLogger(ctx)
	if logger == nil {
		return fmt.Errorf("logger not found in context")
	}

	// Get request ID for correlation
	requestID := shttp.GetRequestID(ctx)
	userID := shttp.GetUserID(ctx)
	clientIP := shttp.GetClientIP(ctx)

	// Log using the logger from context
	logger.Infof(ctx, "Processing home request from user %s at IP %s", userID, clientIP)

	// Write response
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	
	html := `
	<html>
	<head><title>Logger Example</title></head>
	<body>
		<h1>Logger Example</h1>
		<p>Request ID: %s</p>
		<p>User ID: %s</p>
		<p>Client IP: %s</p>
		<h2>Available Routes:</h2>
		<ul>
			<li><a href="/">/</a> - Home page (Info level)</li>
			<li><a href="/debug">/debug</a> - Debug log level example</li>
			<li><a href="/info">/info</a> - Info log level example</li>
			<li><a href="/error">/error</a> - Error log level example</li>
		</ul>
	</body>
	</html>
	`
	
	fmt.Fprintf(w, html, requestID, userID, clientIP)
	return nil
}

// debugLogHandler demonstrates debug level logging
func debugLogHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	logger := shttp.GetLogger(ctx)
	if logger == nil {
		return fmt.Errorf("logger not found in context")
	}

	logger.Debug(ctx, "This is a debug message - useful for detailed troubleshooting")
	
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Debug log entry created - check server logs")
	return nil
}

// infoLogHandler demonstrates info level logging
func infoLogHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	logger := shttp.GetLogger(ctx)
	if logger == nil {
		return fmt.Errorf("logger not found in context")
	}

	logger.Info(ctx, "This is an info message - for normal operational information")
	
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Info log entry created - check server logs")
	return nil
}

// errorLogHandler demonstrates error level logging
func errorLogHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	logger := shttp.GetLogger(ctx)
	if logger == nil {
		return fmt.Errorf("logger not found in context")
	}

	logger.Error(ctx, "This is an error message - something went wrong")
	
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Error log entry created - check server logs")
	return nil
} 