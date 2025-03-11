# shttp

A structured HTTP server wrapper for Go that simplifies HTTP server creation with built-in middleware support, structured routing, and improved error handling.

## Features

- **Simple API**: Clean and intuitive API for creating HTTP servers
- **Middleware Support**: Built-in and custom middleware support
- **Error Handling**: Improved error handling with error returns instead of panics
- **Context-Aware**: First-class context support for better request lifecycle management
- **Graceful Shutdown**: Built-in support for graceful server shutdown
- **TLS Support**: Easy HTTPS server configuration
- **Structured Logging**: Integration with structured logging via slogr
- **Type Safety**: Type-safe handlers and middleware

## Installation

```bash
go get github.com/andres-vara/shttp
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/andres-vara/shttp"
)

func main() {
	// Create a context
	ctx := context.Background()

	// Create a new server with default configuration
	server := shttp.New(ctx, nil)

	// Add a simple route
	server.GET("/hello", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		fmt.Fprintln(w, "Hello, World!")
		return nil
	})

	// Start the server
	log.Println("Starting server at http://localhost:8080")
	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
```

## Handler Signature

The `shttp` package uses a custom handler signature that differs from the standard library:

```go
type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request) error
```

This signature offers several advantages:
1. **Explicit context**: The context is a first-class parameter, not hidden inside the request
2. **Error returns**: Handlers return errors instead of writing them directly to the response
3. **Middleware friendly**: Makes it easier to implement middleware that can handle errors

## Middleware

Middleware is implemented as functions that wrap handlers:

```go
type Middleware func(Handler) Handler
```

Adding middleware to a server:

```go
// Add built-in middleware
server.Use(shttp.RequestIDMiddleware())
server.Use(shttp.LoggingMiddleware(logger))
server.Use(shttp.RecoveryMiddleware(logger))

// Add custom middleware
server.Use(func(next shttp.Handler) shttp.Handler {
    return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
        w.Header().Set("X-Custom-Header", "value")
        return next(ctx, w, r)
    }
})
```

## Built-in Middleware

The `shttp` package includes several built-in middleware:

- **RequestIDMiddleware**: Adds a unique request ID to each request
- **LoggerMiddleware**: Adds a logger to the request context
- **LoggingMiddleware**: Logs request and response details
- **RecoveryMiddleware**: Recovers from panics in handlers
- **CORSMiddleware**: Handles CORS for cross-origin requests
- **TimeoutMiddleware**: Sets a timeout for request processing
- **UserContextMiddleware**: Extracts user information from the request

## Error Handling

The error return value of handlers allows for centralized error handling:

```go
// Global error handling middleware
func errorHandlingMiddleware(next shttp.Handler) shttp.Handler {
    return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
        err := next(ctx, w, r)
        if err == nil {
            return nil
        }

        // Handle error based on type
        switch e := err.(type) {
        case *NotFoundError:
            w.WriteHeader(http.StatusNotFound)
            fmt.Fprintf(w, "Not found: %v", e)
        case *ValidationError:
            w.WriteHeader(http.StatusBadRequest)
            fmt.Fprintf(w, "Invalid input: %v", e)
        default:
            w.WriteHeader(http.StatusInternalServerError)
            fmt.Fprintf(w, "Server error: %v", err)
        }

        return nil
    }
}
```

## Context Values

The package provides helpers for accessing common context values:

```go
// In middleware or handlers
requestID := shttp.GetRequestID(ctx)
userID := shttp.GetUserID(ctx)
clientIP := shttp.GetClientIP(ctx)
```

## Examples

See the [examples](./examples) directory for more complete examples:

- [Basic server](./examples/basic/main.go): Simple HTTP server setup
- [Middleware](./examples/middleware/main.go): Using built-in and custom middleware
- [Error handling](./examples/error-handling/main.go): Centralized error handling
- [TLS](./examples/tls/main.go): Setting up a TLS server

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 
