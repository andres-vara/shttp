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

// LoggerMiddleware adds the logger to the context
func LoggerMiddleware(logger *slogr.Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			// Add logger to context
			ctx = context.WithValue(ctx, LoggerKey, logger)
			
			// Log that we're processing this request (for debugging)
			logger.Debug(ctx, "LoggerMiddleware: Adding logger to context")
			
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

// LoggingMiddleware creates a middleware that logs request details
func LoggingMiddleware(logger *slogr.Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			start := time.Now()

			// Log request with context values
			requestID := GetRequestID(ctx)
			userID := GetUserID(ctx)
			clientIP := GetClientIP(ctx)
			
			// Add structured info to the log entry
			logger.Infof(ctx, "[http.request] Incoming request method=%s path=%s request_id=%s user_id=%s client_ip=%s",
				r.Method, r.URL.Path, requestID, userID, clientIP)

			// Create a response writer that captures the status code
			wrapped := &responseWriter{ResponseWriter: w}

			// Call the next handler
			err := next(ctx, wrapped, r)

			// Log response with context values and timing
			duration := time.Since(start)
			
			if err != nil {
				logger.Errorf(ctx, "[http.response] Request failed method=%s path=%s request_id=%s user_id=%s status=%d duration_ms=%d error=%s",
					r.Method, r.URL.Path, requestID, userID, wrapped.status, duration.Milliseconds(), err.Error())
			} else {
				logger.Infof(ctx, "[http.response Request completed method=%s path=%s request_id=%s user_id=%s status=%d duration_ms=%d",
					r.Method, r.URL.Path, requestID, userID, wrapped.status, duration.Milliseconds())
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

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}