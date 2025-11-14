package main

import (
"context"
"fmt"
"log"
"log/slog"
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

	// Example: Create a server with config-driven logger options
	// This allows you to configure logging format (JSON/Text) and level without code changes
	config := &shttp.Config{
		Addr:           ":8080",
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
		// Specify logger options here instead of creating the logger manually
		LoggerOptions: &slogr.Options{
			// Change these to control logging behavior:
			// - HandlerType: HandlerTypeJSON or HandlerTypeText
			// - Level: slog.LevelDebug, slog.LevelInfo, etc.
			// - AddLevelPrefix: true/false to add level prefix to messages
			Level:          slog.LevelDebug, // Use DEBUG level to see all logs
			HandlerType:    slogr.HandlerTypeJSON, // Use JSON for structured logs
			AddLevelPrefix: true,
		},
	}

	// Create server - it will use LoggerOptions to create the logger
	server := shttp.New(ctx, config)

	// Use the default middleware stack with the server's logger
server.Use(shttp.DefaultMiddlewareStack(server.GetLogger())...)

// Register routes
server.GET("/", homeHandler)
server.GET("/health", healthHandler)
server.GET("/debug", debugHandler)

// Set up a channel to handle shutdown signals
done := make(chan os.Signal, 1)
signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

// Start the server in a goroutine
go func() {
log.Println("Starting server at http://localhost:8080")
log.Println("Server is configured to use JSON logging format with DEBUG level")
log.Println("Try:")
log.Println("  curl http://localhost:8080/")
log.Println("  curl http://localhost:8080/health")
log.Println("  curl http://localhost:8080/debug")
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

// homeHandler shows basic logging with auto-injected attributes
func homeHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
logger := shttp.GetLogger(ctx)
if logger == nil {
return fmt.Errorf("logger not found in context")
}

logger.Info(ctx, "Handling home request")

w.Header().Set("Content-Type", "text/html; charset=utf-8")
fmt.Fprintln(w, `<html>
<head><title>Config-Driven Logger Options Example</title></head>
<body>
<h1>Config-Driven Logger Options Example</h1>
<p>This example demonstrates how to configure logging format and level via Server.Config.LoggerOptions.</p>
<h2>Configuration Benefits</h2>
<ul>
<li><strong>Format Control:</strong> Switch between JSON and Text formats without code changes</li>
<li><strong>Level Control:</strong> Adjust logging level (DEBUG, INFO, WARN, ERROR) via config</li>
<li><strong>Environment-Friendly:</strong> Easy to configure different formats per deployment</li>
<li><strong>Auto-Injected Attributes:</strong> request_id, user_id, client_ip automatically included</li>
</ul>
<h2>Current Setup</h2>
<ul>
<li>Format: JSON (machine-readable)</li>
<li>Level: DEBUG (verbose)</li>
<li>Prefix: Enabled</li>
</ul>
<h2>Try These URLs</h2>
<ul>
<li><a href="/health">GET /health</a></li>
<li><a href="/debug">GET /debug</a></li>
</ul>
<p><strong>Check the server logs:</strong> You'll see JSON-formatted logs with auto-injected request metadata.</p>
	</body>
</html>`)
	return nil
}

// healthHandler shows a simple health check with logging
func healthHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	logger := shttp.GetLogger(ctx)
	if logger == nil {
		return fmt.Errorf("logger not found in context")
	}

	logger.Debug(ctx, "Health check requested")

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "healthy", "timestamp": "%s"}`, time.Now().Format(time.RFC3339))
	return nil
}

// debugHandler demonstrates DEBUG level logging
func debugHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	logger := shttp.GetLogger(ctx)
	if logger == nil {
		return fmt.Errorf("logger not found in context")
	}

	// Log at multiple levels to show how config affects output
	logger.Debug(ctx, "Debug message - detailed info")
	logger.Info(ctx, "Info message - general information")
	logger.Warn(ctx, "Warn message - potential issue")

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message": "Multiple log levels emitted - check server logs"}`)
	return nil
}
