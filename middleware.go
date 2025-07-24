package shttp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

// GetLogger retrieves the logger from the context
func GetLogger(ctx context.Context) *slogr.Logger {
	if logger, ok := ctx.Value(LoggerKey).(*slogr.Logger); ok {
		return logger
	}
	// Return nil if logger is not found in context
	return nil
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
// like request ID, user ID, and client IP, and adds it to the context.
// It assumes that middleware like RequestIDMiddleware and UserContextMiddleware have already been run.
func ContextualLogger(baseLogger *slogr.Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			// Get context values that should have been set by other middleware
			requestID := GetRequestID(ctx)
			userID := GetUserID(ctx)
			clientIP := GetClientIP(ctx)

			// Create a request-scoped logger with the extracted context.
			// NOTE: This assumes the slogr.Logger has a `Logger` field which is a *slog.Logger.
			reqLogger := baseLogger.Logger.With(
				"request_id", requestID,
				"user_id", userID,
				"client_ip", clientIP,
			)

			// Add the new request-scoped logger to the context.
			ctx = context.WithValue(ctx, LoggerKey, &slogr.Logger{Logger: reqLogger})

			// Continue to the next handler in the chain.
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

// LoggingMiddleware creates a middleware that logs request and response details.
// It relies on a logger being present in the context, which is expected to be
// set by the ContextualLogger middleware.
func LoggingMiddleware() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			start := time.Now()

			// Get the logger from the context. If it's not there, we can't log.
			logger := GetLogger(ctx)
			if logger == nil {
				return next(ctx, w, r) // Proceed without logging
			}

			// NOTE: This assumes the logger from context supports structured logging
			// via methods like Info, Error, and With.
			logger.Logger.InfoContext(ctx, "Incoming request", "method", r.Method, "path", r.URL.Path)

			wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			err := next(ctx, w, r)
			duration := time.Since(start)

			// Add response-specific fields to the logger
			resLogger := logger.Logger.With("status", wrapped.status, "duration_ms", duration.Milliseconds())

			if err != nil {
				resLogger.ErrorContext(ctx, "Request failed", "error", err.Error())
			} else {
				resLogger.InfoContext(ctx, "Request completed")
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