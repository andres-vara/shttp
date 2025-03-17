package shttp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andres-vara/slogr"
)


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
			name:   "GET route with path parameters",
			method: http.MethodGet,
			path:   "/test/{param1}",
			setupServer: func(s *Server) {
				s.GET("/test/{param1}", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					param1 := r.PathValue("param1")
					w.Write([]byte(param1))
					return nil
				})
			},
			requestMethod:  http.MethodGet,
			requestPath:    "/test/123",
			wantStatusCode: http.StatusOK,
			wantBody:       "123",
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


