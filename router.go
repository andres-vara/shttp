package shttp

import (
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
	// In Go 1.22+, the standard mux can handle path parameters
	// Let the mux handle the request directly to preserve path parameters
	r.mux.ServeHTTP(w, req)
}

// applyMiddleware wraps the given handler with all middleware
func (r *Router) applyMiddleware(handler Handler) Handler {
	// Apply all middleware in reverse order
	// This creates a processing pipeline where the first middleware in the stack is the outermost wrapper
	result := handler
	for i := len(r.middleware) - 1; i >= 0; i-- {
		result = r.middleware[i](result)
	}
	return result
}

// GET registers a GET route handler
func (r *Router) GET(path string, handler Handler) {
	r.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		// Create a context from the request
		ctx := req.Context()
		
		// Apply middleware to the handler
		handlerWithMiddleware := r.applyMiddleware(handler)
		
		if err := handlerWithMiddleware(ctx, w, req); err != nil {
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
		
		// Create a context from the request
		ctx := req.Context()
		
		// Apply middleware to the handler
		handlerWithMiddleware := r.applyMiddleware(handler)
		
		if err := handlerWithMiddleware(ctx, w, req); err != nil {
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
		
		// Create a context from the request
		ctx := req.Context()
		
		// Apply middleware to the handler
		handlerWithMiddleware := r.applyMiddleware(handler)
		
		if err := handlerWithMiddleware(ctx, w, req); err != nil {
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
		
		// Create a context from the request
		ctx := req.Context()
		
		// Apply middleware to the handler
		handlerWithMiddleware := r.applyMiddleware(handler)
		
		if err := handlerWithMiddleware(ctx, w, req); err != nil {
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
		
		// Create a context from the request
		ctx := req.Context()
		
		// Apply middleware to the handler
		handlerWithMiddleware := r.applyMiddleware(handler)
		
		if err := handlerWithMiddleware(ctx, w, req); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

// Use adds middleware to the router
func (r *Router) Use(middleware Middleware) {
	r.middleware = append(r.middleware, middleware)
} 