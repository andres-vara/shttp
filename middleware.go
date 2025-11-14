package shttp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/andres-vara/slogr"
)

// Handler is a function that handles HTTP requests
type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

// Middleware is a function that wraps a handler
type Middleware func(Handler) Handler

// ContextKey is a type for context keys to avoid collisions
type ContextKey string

const (
	// RequestIDKey is the context key for the request ID
	RequestIDKey ContextKey = "request_id"
	// UserIDKey is the context key for the user ID
	UserIDKey ContextKey = "user_id"
	// ClientIPKey is the context key for the client IP
	ClientIPKey ContextKey = "client_ip"
	// LoggerKey is the context key for the logger
	LoggerKey ContextKey = "logger"
)

// GetRequestID retrieves the request ID from the context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// GetUserID retrieves the user ID from the context
func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(UserIDKey).(string); ok {
		return id
	}
	return ""
}

// GetClientIP retrieves the client IP from the context
func GetClientIP(ctx context.Context) string {
	if ip, ok := ctx.Value(ClientIPKey).(string); ok {
		return ip
	}
	return ""
}

// GetLogger retrieves the logger from the context.
// Prefers slogr.FromContext for unified access across packages.
func GetLogger(ctx context.Context) *slogr.Logger {
	// Try slogr's context key first for unified access
	if logger := slogr.FromContext(ctx); logger != nil {
		return logger
	}
	// Fallback to shttp's internal key for backward compatibility
	if logger, ok := ctx.Value(LoggerKey).(*slogr.Logger); ok {
		return logger
	}
	return nil
}

// WithLogger returns a new context with the logger added, using slogr's unified key.
// This delegates to slogr.WithLogger for consistency across packages.
func WithLogger(ctx context.Context, logger *slogr.Logger) context.Context {
	return slogr.WithLogger(ctx, logger)
}

// generates a random request ID
func generateRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// If we can't generate a random ID, use timestamp as fallback
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(bytes)
}

// RequestIDMiddleware adds a unique request ID to the context
func RequestIDMiddleware() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			// Generate a unique request ID
			requestID := generateRequestID()

			// Add to both context and response headers
			ctx = context.WithValue(ctx, RequestIDKey, requestID)
			w.Header().Set("X-Request-ID", requestID)

			// Extract client IP (simplified)
			clientIP := r.RemoteAddr
			if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
				clientIP = forwardedFor
			}
			ctx = context.WithValue(ctx, ClientIPKey, clientIP)

			// Continue with request handling
			return next(ctx, w, r)
		}
	}
}

// ContextualLogger creates a request-scoped logger with contextual information
// (request ID, user ID, client IP) as structured attributes and adds it to the context.
// It assumes that middleware like RequestIDMiddleware and UserContextMiddleware have already been run.
func ContextualLogger(baseLogger *slogr.Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			// Inject request metadata as structured attributes
			ctx = slogr.WithAttrs(ctx,
				slog.String("request_id", GetRequestID(ctx)),
				slog.String("user_id", GetUserID(ctx)),
				slog.String("client_ip", GetClientIP(ctx)),
			)
			// Add logger to context using unified slogr key
			ctx = slogr.WithLogger(ctx, baseLogger)
			return next(ctx, w, r)
		}
	}
}

// UserContextMiddleware extracts user info from the request (e.g., from JWT)
// and adds it to the context
func UserContextMiddleware() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			// This is a simplified example - in a real app, you'd extract the user ID
			// from JWT or session

			// For example, perhaps from Authorization header
			userID := "anonymous"
			if authHeader := r.Header.Get("Authorization"); authHeader != "" {
				// In reality, you'd validate and extract user ID from the token
				userID = "authenticated-user"
			}

			ctx = context.WithValue(ctx, UserIDKey, userID)

			return next(ctx, w, r)
		}
	}
}

// LoggerMiddleware attaches the provided logger into the request context.
// This is a convenience wrapper around ContextualLogger to match historical
// usage where callers pass the logger explicitly.
func LoggerMiddleware(logger *slogr.Logger) Middleware {
	return ContextualLogger(logger)
}

// LoggingMiddleware creates a middleware that logs request and response details.
// If a non-nil logger is provided it will be used directly; otherwise the
// middleware will try to obtain a logger from the request context.
func LoggingMiddleware(logger *slogr.Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			start := time.Now()
			var l *slogr.Logger
			if logger != nil {
				l = logger
			} else {
				l = GetLogger(ctx)
			}
			if l == nil {
				// No logger available, proceed without logging
				return next(ctx, w, r)
			}
			// Log a request entry with contextual fields
			l.Infof(ctx, "[http.request] method=%s path=%s request_id=%s user_id=%s client_ip=%s", r.Method, r.URL.Path, GetRequestID(ctx), GetUserID(ctx), GetClientIP(ctx))

			err := next(ctx, w, r)
			duration := time.Since(start)

			// Log a response entry with status/duration and optional error
			if err != nil {
				l.Errorf(ctx, "[http.response] method=%s path=%s request_id=%s user_id=%s client_ip=%s error=%v duration_ms=%d", r.Method, r.URL.Path, GetRequestID(ctx), GetUserID(ctx), GetClientIP(ctx), err, duration.Milliseconds())
			} else {
				// try to obtain status code if responseWriter wrapped this (best-effort)
				status := http.StatusOK
				if rw, ok := w.(*responseWriter); ok && rw.status != 0 {
					status = rw.status
				}
				l.Infof(ctx, "[http.response] method=%s path=%s request_id=%s user_id=%s client_ip=%s status=%d duration_ms=%d", r.Method, r.URL.Path, GetRequestID(ctx), GetUserID(ctx), GetClientIP(ctx), status, duration.Milliseconds())
			}
			return err
		}
	}
}

// RecoveryMiddleware creates a middleware that recovers from panics
func RecoveryMiddleware(logger *slogr.Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (err error) {
			defer func() {
				if rec := recover(); rec != nil {
					// Log the panic with context values
					requestID := GetRequestID(ctx)
					userID := GetUserID(ctx)

					logger.Errorf(ctx, "[http.panic] Recovered from panic: %v, request_id: %s, user_id: %s, method: %s, path: %s",
						rec,
						requestID,
						userID,
						r.Method,
						r.URL.Path)

					// Return a 500 error
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					err = fmt.Errorf("panic: %v", rec)
				}
			}()
			return next(ctx, w, r)
		}
	}
}

// CORSMiddleware creates a middleware that handles CORS
func CORSMiddleware(allowedOrigins []string) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Max-Age", "3600")
				w.WriteHeader(http.StatusOK)
				return nil
			}

			// Add CORS headers to response
			origin := r.Header.Get("Origin")
			for _, allowed := range allowedOrigins {
				if allowed == "*" || allowed == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}

			return next(ctx, w, r)
		}
	}
}

// TimeoutMiddleware creates a middleware that adds a timeout to the request context
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			return next(ctx, w, r)
		}
	}
}

// responseWriter wraps http.ResponseWriter to capture status and prevent multiple header writes.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *responseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
	w.wroteHeader = true
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// DefaultMiddlewareStack returns a recommended middleware stack for typical HTTP services.
// It includes: request ID generation, user context extraction, contextual logger injection
// with request attributes, request/response logging, and panic recovery.
// The stack is ordered for optimal request flow and logging visibility.
func DefaultMiddlewareStack(logger *slogr.Logger) []Middleware {
	return []Middleware{
		RequestIDMiddleware(),
		UserContextMiddleware(),
		ContextualLogger(logger),
		LoggingMiddleware(logger),
		RecoveryMiddleware(logger),
	}
}
