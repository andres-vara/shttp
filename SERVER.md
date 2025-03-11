# shttp Server Implementation

This document provides a technical overview of how the server implementation works in the `shttp` package.

## Server Architecture

The `shttp` server is designed with a layered architecture:

1. **Server Layer** - Manages the HTTP server lifecycle
2. **Router Layer** - Handles request routing and middleware
3. **Handler Layer** - Processes individual requests

## Key Components

### Server

The `Server` struct wraps the standard library's `http.Server` and adds:

- Structured configuration
- Context-aware operations
- Built-in routing
- Method-specific handler registration
- Middleware support

### Router

The `Router` implements `http.Handler` and provides:

- Method-specific routing (GET, POST, PUT, DELETE, PATCH)
- Middleware stack management
- Error handling for handlers

### Handler & Middleware

The custom handler signature improves on the standard library:

```go
// Handler is a function that handles HTTP requests
type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

// Middleware is a function that wraps a handler
type Middleware func(Handler) Handler
```

## Middleware Implementation

Middleware in shttp uses function composition:

1. Middleware functions are registered with `Use()`
2. During request handling, middleware is applied in reverse order (last-in, first-out)
3. This creates a "pipeline" or "onion" model where each middleware wraps around the next

```go
// Apply all middleware
for i := len(r.middleware) - 1; i >= 0; i-- {
    handlerFunc = r.middleware[i](handlerFunc)
}
```

This approach allows middleware to perform operations before and after the handler executes, and creates a natural flow where:

- Request processing flows from outermost to innermost middleware
- Response processing flows from innermost to outermost middleware

## Error Handling

Unlike the standard library, handlers return errors explicitly, which:

1. Separates error handling from successful response generation
2. Enables centralized error handling at the router level
3. Allows middleware to catch and process errors from inner handlers
4. Makes testing easier by allowing direct assertion on returned errors

## HTTP Method Handling

The router's method handlers (GET, POST, etc.) handle method validation and provide a consistent error handling approach:

```go
r.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
    if req.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    if err := handler(req.Context(), w, req); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
})
```

## Context Handling

The server preserves context throughout the request lifecycle:
- The server has a root context passed during creation
- Requests get their own context derived from the incoming request
- Middleware can enrich the context with additional values 