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

// Handle registers a handler for the given method and path.
func (r *Router) Handle(method, path string, handler Handler) {
	r.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != method {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := req.Context()
		handlerWithMiddleware := r.applyMiddleware(handler)

		// Create a new response writer to track whether the header has been written.
		rw := &responseWriter{ResponseWriter: w}

		// Call the handler with the wrapped response writer.
		if err := handlerWithMiddleware(ctx, rw, req); err != nil {
			// If the header has not been written, write the error to the response.
			if !rw.wroteHeader {
				if httpErr, ok := err.(HTTPError); ok {
					http.Error(w, httpErr.Message, httpErr.StatusCode)
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}
		}
	})
}

// GET registers a GET route handler
func (r *Router) GET(path string, handler Handler) {
	r.Handle(http.MethodGet, path, handler)
}

// POST registers a POST route handler
func (r *Router) POST(path string, handler Handler) {
	r.Handle(http.MethodPost, path, handler)
}

// PUT registers a PUT route handler
func (r *Router) PUT(path string, handler Handler) {
	r.Handle(http.MethodPut, path, handler)
}

// DELETE registers a DELETE route handler
func (r *Router) DELETE(path string, handler Handler) {
	r.Handle(http.MethodDelete, path, handler)
}

// PATCH registers a PATCH route handler
func (r *Router) PATCH(path string, handler Handler) {
	r.Handle(http.MethodPatch, path, handler)
}

// ANY registers a handler for all HTTP methods on a path.
// Internally it registers a single handler without method filtering.
func (r *Router) ANY(path string, handler Handler) {
	r.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		handlerWithMiddleware := r.applyMiddleware(handler)

		// Wrap the response writer to track header writes.
		rw := &responseWriter{ResponseWriter: w}
		if err := handlerWithMiddleware(ctx, rw, req); err != nil {
			if !rw.wroteHeader {
				if httpErr, ok := err.(HTTPError); ok {
					http.Error(w, httpErr.Message, httpErr.StatusCode)
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}
		}
	})
}

// Use adds middleware to the router
func (r *Router) Use(middleware ...Middleware) {
	r.middleware = append(r.middleware, middleware...)
} 
