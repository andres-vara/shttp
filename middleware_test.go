package shttp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andres-vara/slogr"
)

// Helper function to create a simple handler that writes a message to the response
func simpleHandler(message string) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		w.Write([]byte(message))
		return nil
	}
}

// Helper function to create a handler that returns an error
func errorHandler(errorMsg string) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return fmt.Errorf("[error] %s", errorMsg)
	}
}

// Helper function to execute a middleware test
func executeMiddlewareTest(_ *testing.T, middleware Middleware, handler Handler, req *http.Request) *httptest.ResponseRecorder {
	// Create a response recorder
	w := httptest.NewRecorder()

	// Apply the middleware to the handler
	wrappedHandler := middleware(handler)

	// Execute the wrapped handler
	err := wrappedHandler(req.Context(), w, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	return w
}

func TestRequestIDMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		handler        Handler
		wantStatusCode int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Adds request ID header",
			handler:        simpleHandler("test"),
			wantStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Check that the X-Request-ID header was set
				requestID := w.Header().Get("X-Request-ID")
				if requestID == "" {
					t.Error("RequestIDMiddleware did not set X-Request-ID header")
				}

				// Check that the request ID is the expected format (hex string)
				if len(requestID) != 32 { // 16 bytes as hex string is 32 chars
					t.Errorf("Request ID has unexpected length: got %d, want 32", len(requestID))
				}
			},
		},
		{
			name: "Adds client IP to context",
			handler: func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				// Extract the client IP from the context
				clientIP := GetClientIP(ctx)
				w.Write([]byte(clientIP))
				return nil
			},
			wantStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				// The body should contain the client IP
				clientIP := w.Body.String()
				if clientIP == "" {
					t.Error("RequestIDMiddleware did not add client IP to context")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Execute the test
			w := executeMiddlewareTest(t, RequestIDMiddleware(), tt.handler, req)

			// Check the status code
			if w.Code != tt.wantStatusCode {
				t.Errorf("Status code = %v, want %v", w.Code, tt.wantStatusCode)
			}

			// Run the response check
			tt.checkResponse(t, w)
		})
	}
}

func TestUserContextMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		setupRequest   func(*http.Request)
		handler        Handler
		wantStatusCode int
		wantUserID     string
	}{
		{
			name: "Anonymous user without Authorization",
			setupRequest: func(r *http.Request) {
				// No Authorization header
			},
			handler: func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				userID := GetUserID(ctx)
				w.Write([]byte(userID))
				return nil
			},
			wantStatusCode: http.StatusOK,
			wantUserID:     "anonymous",
		},
		{
			name: "Authenticated user with Authorization",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer token123")
			},
			handler: func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				userID := GetUserID(ctx)
				w.Write([]byte(userID))
				return nil
			},
			wantStatusCode: http.StatusOK,
			wantUserID:     "authenticated-user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setupRequest(req)

			// Execute the test
			w := executeMiddlewareTest(t, UserContextMiddleware(), tt.handler, req)

			// Check the status code
			if w.Code != tt.wantStatusCode {
				t.Errorf("Status code = %v, want %v", w.Code, tt.wantStatusCode)
			}

			// Check the user ID
			if w.Body.String() != tt.wantUserID {
				t.Errorf("User ID = %q, want %q", w.Body.String(), tt.wantUserID)
			}
		})
	}
}

func TestLoggerMiddleware(t *testing.T) {
	// Create a logger that writes to a string builder
	var logOutput strings.Builder

	logger := slogr.New(&logOutput, &slogr.Options{
		Level:       slog.LevelDebug,
		HandlerType: slogr.HandlerTypeJSON,
	})

	tests := []struct {
		name            string
		handler         Handler
		wantStatusCode  int
		wantLogContains string
	}{
		{
			name: "Adds logger to context",
			handler: func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				// Get the logger from context and log something
				if ctxLogger, ok := ctx.Value(LoggerKey).(*slogr.Logger); ok && ctxLogger != nil {
					ctxLogger.Info(ctx, "Test log from handler")
					w.Write([]byte("logged"))
					return nil
				}
				return fmt.Errorf("logger not found in context")
			},
			wantStatusCode:  http.StatusOK,
			wantLogContains: "Test log from handler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the log output
			logOutput.Reset()

			// Create a test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Execute the test
			w := executeMiddlewareTest(t, LoggerMiddleware(logger), tt.handler, req)

			// Check the status code
			if w.Code != tt.wantStatusCode {
				t.Errorf("Status code = %v, want %v", w.Code, tt.wantStatusCode)
			}

			// Check the log output
			logStr := logOutput.String()
			if !strings.Contains(logStr, tt.wantLogContains) {
				t.Errorf("Log output does not contain %q: %q", tt.wantLogContains, logStr)
			}
		})
	}
}

func TestLoggingMiddleware(t *testing.T) {
	// Create a logger that writes to a string builder
	var logOutput strings.Builder
	logger := slogr.New(&logOutput, slogr.DefaultOptions())

	tests := []struct {
		name               string
		setupContext       func(context.Context) context.Context
		handler            Handler
		wantStatusCode     int
		wantLogContains    []string
		wantLogNotContains []string
	}{
		{
			name: "Logs successful request",
			setupContext: func(ctx context.Context) context.Context {
				ctx = context.WithValue(ctx, RequestIDKey, "test-request-id")
				ctx = context.WithValue(ctx, UserIDKey, "test-user-id")
				ctx = context.WithValue(ctx, ClientIPKey, "127.0.0.1")
				return ctx
			},
			handler:        simpleHandler("success"),
			wantStatusCode: http.StatusOK,
			wantLogContains: []string{
				"[http.request]",
				"[http.response",
				"method=GET",
				"path=/test",
				"request_id=test-request-id",
				"user_id=test-user-id",
				"client_ip=127.0.0.1",
				"status=200",
			},
			wantLogNotContains: []string{
				"error=",
			},
		},
		{
			name: "Logs failed request",
			setupContext: func(ctx context.Context) context.Context {
				ctx = context.WithValue(ctx, RequestIDKey, "test-request-id")
				ctx = context.WithValue(ctx, UserIDKey, "test-user-id")
				ctx = context.WithValue(ctx, ClientIPKey, "127.0.0.1")
				return ctx
			},
			handler:        errorHandler("test error"),
			wantStatusCode: http.StatusInternalServerError,
			wantLogContains: []string{
				"[http.request]",
				"[http.response]",
				"method=GET",
				"path=/test",
				"request_id=test-request-id",
				"user_id=test-user-id",
				"client_ip=127.0.0.1",
				"error=[error] test error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the log output
			logOutput.Reset()

			// Create a test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Set up the context
			ctx := tt.setupContext(req.Context())
			req = req.WithContext(ctx)

			// Execute the test
			w := executeMiddlewareTest(t, LoggingMiddleware(logger), tt.handler, req)

			// Check the status code
			if w.Code != tt.wantStatusCode {
				t.Errorf("Status code = %v, want %v", w.Code, tt.wantStatusCode)
			}

			// Check the log output
			logStr := logOutput.String()
			for _, wantStr := range tt.wantLogContains {
				if !strings.Contains(logStr, wantStr) {
					t.Errorf("Log output does not contain %q: %q", wantStr, logStr)
				}
			}

			// Check that unwanted strings are not in the log
			if tt.wantLogNotContains != nil {
				for _, unwantedStr := range tt.wantLogNotContains {
					if strings.Contains(logStr, unwantedStr) {
						t.Errorf("Log output should not contain %q: %q", unwantedStr, logStr)
					}
				}
			}
		})
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	// Create a logger that writes to a string builder
	var logOutput strings.Builder
	logger := slogr.New(&logOutput, slogr.DefaultOptions())

	tests := []struct {
		name            string
		setupContext    func(context.Context) context.Context
		handler         Handler
		wantStatusCode  int
		wantLogContains []string
	}{
		{
			name: "Recovers from panic",
			setupContext: func(ctx context.Context) context.Context {
				ctx = context.WithValue(ctx, RequestIDKey, "test-request-id")
				ctx = context.WithValue(ctx, UserIDKey, "test-user-id")
				return ctx
			},
			handler: func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				panic("test panic")
			},
			wantStatusCode: http.StatusInternalServerError,
			wantLogContains: []string{
				"[http.panic]",
				"Recovered from panic",
				"test panic",
				"request_id: test-request-id",
				"user_id: test-user-id",
			},
		},
		{
			name: "Does not interfere with normal requests",
			setupContext: func(ctx context.Context) context.Context {
				return ctx
			},
			handler:         simpleHandler("normal"),
			wantStatusCode:  http.StatusOK,
			wantLogContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the log output
			logOutput.Reset()

			// Create a test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Set up the context
			ctx := tt.setupContext(req.Context())
			req = req.WithContext(ctx)

			// Execute the test
			w := executeMiddlewareTest(t, RecoveryMiddleware(logger), tt.handler, req)

			// Check the status code
			if w.Code != tt.wantStatusCode {
				t.Errorf("Status code = %v, want %v", w.Code, tt.wantStatusCode)
			}

			// Check the log output
			logStr := logOutput.String()
			for _, wantStr := range tt.wantLogContains {
				if !strings.Contains(logStr, wantStr) {
					t.Errorf("Log output does not contain %q: %q", wantStr, logStr)
				}
			}
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins []string
		setupRequest   func(*http.Request)
		handler        Handler
		wantStatusCode int
		wantHeaders    map[string]string
	}{
		{
			name:           "OPTIONS request",
			allowedOrigins: []string{"https://example.com"},
			setupRequest: func(r *http.Request) {
				r.Method = http.MethodOptions
				r.Header.Set("Origin", "https://example.com")
			},
			handler:        simpleHandler("test"),
			wantStatusCode: http.StatusOK,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, PATCH, OPTIONS",
				"Access-Control-Allow-Headers": "Content-Type, Authorization",
				"Access-Control-Max-Age":       "3600",
			},
		},
		{
			name:           "Request with allowed origin",
			allowedOrigins: []string{"https://example.com"},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Origin", "https://example.com")
			},
			handler:        simpleHandler("test"),
			wantStatusCode: http.StatusOK,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin": "https://example.com",
			},
		},
		{
			name:           "Request with disallowed origin",
			allowedOrigins: []string{"https://example.com"},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Origin", "https://evil.com")
			},
			handler:        simpleHandler("test"),
			wantStatusCode: http.StatusOK,
			wantHeaders:    map[string]string{},
		},
		{
			name:           "Wildcard origin",
			allowedOrigins: []string{"*"},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Origin", "https://any-domain.com")
			},
			handler:        simpleHandler("test"),
			wantStatusCode: http.StatusOK,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin": "https://any-domain.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setupRequest(req)

			// Execute the test
			w := executeMiddlewareTest(t, CORSMiddleware(tt.allowedOrigins), tt.handler, req)

			// Check the status code
			if w.Code != tt.wantStatusCode {
				t.Errorf("Status code = %v, want %v", w.Code, tt.wantStatusCode)
			}

			// Check the headers
			for k, v := range tt.wantHeaders {
				if got := w.Header().Get(k); got != v {
					t.Errorf("Header %q = %q, want %q", k, got, v)
				}
			}
		})
	}
}

func TestTimeoutMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		timeout        time.Duration
		handler        Handler
		wantStatusCode int
		wantTimeout    bool
	}{
		{
			name:    "Short timeout with quick handler",
			timeout: 100 * time.Millisecond,
			handler: func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(10 * time.Millisecond):
					w.Write([]byte("success"))
					return nil
				}
			},
			wantStatusCode: http.StatusOK,
			wantTimeout:    false,
		},
		{
			name:    "Short timeout with slow handler",
			timeout: 10 * time.Millisecond,
			handler: func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(50 * time.Millisecond):
					w.Write([]byte("success"))
					return nil
				}
			},
			wantStatusCode: http.StatusInternalServerError,
			wantTimeout:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Execute the test
			w := executeMiddlewareTest(t, TimeoutMiddleware(tt.timeout), tt.handler, req)

			// Check the status code
			if tt.wantTimeout {
				if w.Code != http.StatusInternalServerError {
					t.Errorf("Status code = %v, want %v", w.Code, http.StatusInternalServerError)
				}
				if !strings.Contains(w.Body.String(), "context deadline exceeded") {
					t.Errorf("Error message should mention timeout, got: %q", w.Body.String())
				}
			} else {
				if w.Code != tt.wantStatusCode {
					t.Errorf("Status code = %v, want %v", w.Code, tt.wantStatusCode)
				}
			}
		})
	}
}

func TestResponseWriter(t *testing.T) {
	tests := []struct {
		name           string
		writeStatus    int
		writeBody      string
		wantStatusCode int
	}{
		{
			name:           "Explicit status code",
			writeStatus:    http.StatusCreated,
			writeBody:      "Created",
			wantStatusCode: http.StatusCreated,
		},
		{
			name:           "Default status code",
			writeStatus:    0, // Don't explicitly set the status
			writeBody:      "OK",
			wantStatusCode: http.StatusOK, // Should default to 200 OK
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test ResponseWriter
			w := httptest.NewRecorder()
			rw := &responseWriter{ResponseWriter: w}

			// Set the status if specified
			if tt.writeStatus != 0 {
				rw.WriteHeader(tt.writeStatus)
			}

			// Write the body
			rw.Write([]byte(tt.writeBody))

			// Check the status code
			if rw.status != tt.wantStatusCode {
				t.Errorf("responseWriter.status = %v, want %v", rw.status, tt.wantStatusCode)
			}

			if w.Code != tt.wantStatusCode {
				t.Errorf("ResponseWriter.Code = %v, want %v", w.Code, tt.wantStatusCode)
			}

			// Check the body
			if w.Body.String() != tt.writeBody {
				t.Errorf("ResponseWriter.Body = %q, want %q", w.Body.String(), tt.writeBody)
			}
		})
	}
}
