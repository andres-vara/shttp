package shttp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/andres-vara/slogr"
)

func TestNew(t *testing.T) {
	// Table-driven test cases
	tests := []struct {
		name       string
		config     *Config
		wantAddr   string
		wantLogger bool
	}{
		{
			name:       "Default config",
			config:     nil,
			wantAddr:   ":8080",
			wantLogger: true,
		},
		{
			name: "Custom config",
			config: &Config{
				Addr:           ":9090",
				ReadTimeout:    5 * time.Second,
				WriteTimeout:   5 * time.Second,
				IdleTimeout:    60 * time.Second,
				MaxHeaderBytes: 1 << 10, // 1KB
				Logger:         slogr.New(os.Stdout, slogr.DefaultOptions()),
			},
			wantAddr:   ":9090",
			wantLogger: true,
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			server := New(ctx, tt.config)

			// Check that the server was created correctly
			if server.server.Addr != tt.wantAddr {
				t.Errorf("New() server.Addr = %v, want %v", server.server.Addr, tt.wantAddr)
			}

			// Check that the router was created
			if server.router == nil {
				t.Error("New() server.router is nil")
			}

			// Check that the logger was set
			if (server.logger != nil) != tt.wantLogger {
				t.Errorf("New() server.logger = %v, want %v", server.logger != nil, tt.wantLogger)
			}
		})
	}
}

func TestServerRouting(t *testing.T) {
	// Test cases for HTTP method registration
	tests := []struct {
		name           string
		method         string
		path           string
		setupServer    func(*Server)
		requestMethod  string
		requestPath    string
		wantStatusCode int
		wantBody       string
	}{
		{
			name:   "GET route success",
			method: http.MethodGet,
			path:   "/test",
			setupServer: func(s *Server) {
				s.GET("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("GET success"))
					return nil
				})
			},
			requestMethod:  http.MethodGet,
			requestPath:    "/test",
			wantStatusCode: http.StatusOK,
			wantBody:       "GET success",
		},
		{
			name:   "GET route method not allowed",
			method: http.MethodGet,
			path:   "/test",
			setupServer: func(s *Server) {
				s.GET("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("GET success"))
					return nil
				})
			},
			requestMethod:  http.MethodPost,
			requestPath:    "/test",
			wantStatusCode: http.StatusMethodNotAllowed,
			wantBody:       "Method not allowed\n",
		},
		{
			name:   "POST route success",
			method: http.MethodPost,
			path:   "/test",
			setupServer: func(s *Server) {
				s.POST("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("POST success"))
					return nil
				})
			},
			requestMethod:  http.MethodPost,
			requestPath:    "/test",
			wantStatusCode: http.StatusOK,
			wantBody:       "POST success",
		},
		{
			name:   "PUT route success",
			method: http.MethodPut,
			path:   "/test",
			setupServer: func(s *Server) {
				s.PUT("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("PUT success"))
					return nil
				})
			},
			requestMethod:  http.MethodPut,
			requestPath:    "/test",
			wantStatusCode: http.StatusOK,
			wantBody:       "PUT success",
		},
		{
			name:   "DELETE route success",
			method: http.MethodDelete,
			path:   "/test",
			setupServer: func(s *Server) {
				s.DELETE("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("DELETE success"))
					return nil
				})
			},
			requestMethod:  http.MethodDelete,
			requestPath:    "/test",
			wantStatusCode: http.StatusOK,
			wantBody:       "DELETE success",
		},
		{
			name:   "PATCH route success",
			method: http.MethodPatch,
			path:   "/test",
			setupServer: func(s *Server) {
				s.PATCH("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("PATCH success"))
					return nil
				})
			},
			requestMethod:  http.MethodPatch,
			requestPath:    "/test",
			wantStatusCode: http.StatusOK,
			wantBody:       "PATCH success",
		},
		{
			name:   "Route handler returning error",
			method: http.MethodGet,
			path:   "/error",
			setupServer: func(s *Server) {
				s.GET("/error", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					return errors.New("handler error")
				})
			},
			requestMethod:  http.MethodGet,
			requestPath:    "/error",
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "handler error\n",
		},
		{
			name:   "Route not found",
			method: http.MethodGet,
			path:   "/test",
			setupServer: func(s *Server) {
				s.GET("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("GET success"))
					return nil
				})
			},
			requestMethod:  http.MethodGet,
			requestPath:    "/not-found",
			wantStatusCode: http.StatusNotFound,
			wantBody:       "404 page not found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new server for each test
			ctx := context.Background()
			logger := slogr.New(io.Discard, slogr.DefaultOptions()) // Discard logs for tests
			config := &Config{
				Addr:   ":0", // Use any available port
				Logger: logger,
			}
			server := New(ctx, config)

			// Set up the server with the test route
			tt.setupServer(server)

			// Create a test request
			req := httptest.NewRequest(tt.requestMethod, tt.requestPath, nil)
			w := httptest.NewRecorder()

			// Serve the request
			server.router.ServeHTTP(w, req)

			// Check the response
			if w.Code != tt.wantStatusCode {
				t.Errorf("Status code = %v, want %v", w.Code, tt.wantStatusCode)
			}

			if w.Body.String() != tt.wantBody {
				t.Errorf("Body = %q, want %q", w.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestRouterMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		setupRouter    func(*Router)
		requestMethod  string
		requestPath    string
		wantStatusCode int
		wantBody       string
		wantHeaders    map[string]string
	}{
		{
			name: "Apply single middleware",
			setupRouter: func(r *Router) {
				// Add a simple middleware that adds a header
				r.Use(func(next Handler) Handler {
					return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
						w.Header().Set("X-Test", "middleware-applied")
						return next(ctx, w, r)
					}
				})

				// Add a route
				r.GET("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("test"))
					return nil
				})
			},
			requestMethod:  http.MethodGet,
			requestPath:    "/test",
			wantStatusCode: http.StatusOK,
			wantBody:       "test",
			wantHeaders: map[string]string{
				"X-Test": "middleware-applied",
			},
		},
		{
			name: "Apply multiple middleware in correct order",
			setupRouter: func(r *Router) {
				// First middleware
				r.Use(func(next Handler) Handler {
					return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
						w.Header().Set("X-Order", "first")
						return next(ctx, w, r)
					}
				})

				// Second middleware (should override the first)
				r.Use(func(next Handler) Handler {
					return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
						w.Header().Set("X-Order", "second")
						return next(ctx, w, r)
					}
				})

				// Add a route
				r.GET("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("test"))
					return nil
				})
			},
			requestMethod:  http.MethodGet,
			requestPath:    "/test",
			wantStatusCode: http.StatusOK,
			wantBody:       "test",
			wantHeaders: map[string]string{
				"X-Order": "second", // The second middleware should be applied last
			},
		},
		{
			name: "Middleware returning error",
			setupRouter: func(r *Router) {
				// Add a middleware that returns an error
				r.Use(func(next Handler) Handler {
					return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
						return errors.New("middleware error")
					}
				})

				// Add a route
				r.GET("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					w.Write([]byte("test"))
					return nil
				})
			},
			requestMethod:  http.MethodGet,
			requestPath:    "/test",
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "middleware error\n",
			wantHeaders:    map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new router for each test
			router := NewRouter()

			// Set up the router with middleware and routes
			tt.setupRouter(router)

			// Create a test request
			req := httptest.NewRequest(tt.requestMethod, tt.requestPath, nil)
			w := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(w, req)

			// Check the status code
			if w.Code != tt.wantStatusCode {
				t.Errorf("Status code = %v, want %v", w.Code, tt.wantStatusCode)
			}

			// Check the body
			if w.Body.String() != tt.wantBody {
				t.Errorf("Body = %q, want %q", w.Body.String(), tt.wantBody)
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