package shttp

import (
	"context"
	"net/http"
)

// Router handles HTTP routing
type Router struct {
	// The underlying http.ServeMux
	mux *http.ServeMux

	// Middleware stack
	middleware []Middleware
}

// NewRouter creates a new router
func NewRouter() *Router {
	return &Router{
		mux: http.NewServeMux(),
	}
}

// ServeHTTP implements the http.Handler interface
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Find the handler for the request
	handler, _ := r.mux.Handler(req)
	if handler == nil {
		http.NotFound(w, req)
		return
	}

	// Create a handler function that calls the original handler
	handlerFunc := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		handler.ServeHTTP(w, r)
		return nil
	}

	// Apply all middleware
	// uses function composition to create a processing pipeline for HTTP requests
	for i := len(r.middleware) - 1; i >= 0; i-- {
		// each middleware function takes a handler and returns a new handler
		// the middleware functions are applied in reverse order, so the outermost middleware is applied first
		handlerFunc = r.middleware[i](handlerFunc)
	}

	// Execute the handler
	if err := handlerFunc(req.Context(), w, req); err != nil {
		// Handle error
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// GET registers a GET route handler
func (r *Router) GET(path string, handler Handler) {
	r.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := handler(req.Context(), w, req); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

// POST registers a POST route handler
func (r *Router) POST(path string, handler Handler) {
	r.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := handler(req.Context(), w, req); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

// PUT registers a PUT route handler
func (r *Router) PUT(path string, handler Handler) {
	r.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := handler(req.Context(), w, req); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

// DELETE registers a DELETE route handler
func (r *Router) DELETE(path string, handler Handler) {
	r.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := handler(req.Context(), w, req); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

// PATCH registers a PATCH route handler
func (r *Router) PATCH(path string, handler Handler) {
	r.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPatch {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := handler(req.Context(), w, req); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

// Use adds middleware to the router
func (r *Router) Use(middleware Middleware) {
	r.middleware = append(r.middleware, middleware)
} 