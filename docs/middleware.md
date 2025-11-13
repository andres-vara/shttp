# Middleware Example — shttp

This document provides a longer example showing how to wire request ID and structured logging middleware together with handlers that read path parameters and use the request-scoped logger.

```go
package main

import (
  "context"
  "log"
  "net/http"

  "github.com/andres-vara/shttp"
  "github.com/andres-vara/slogr"
)

func main() {
  // Create a package-level logger (slogr wraps slog)
  logger := slogr.New()

  r := shttp.NewRouter()

  // Apply middleware: request ID then logger. Order matters (request ID first so
  // the logger can include it in logs produced during the request).
  r.Use(
    shttp.RequestIDMiddleware(),
    shttp.LoggerMiddleware(logger),
  )

  // A handler that uses the request-scoped logger
  r.GET("/users/{id}", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    id := shttp.PathValue(r, "id")
    // Retrieve the logger from context (LoggerMiddleware places it there)
    lg := shttp.LoggerFromContext(ctx)
    lg.Info("handling request", "user_id", id)
    _, _ = w.Write([]byte("user: " + id))
    return nil
  })

  log.Println("starting server :8080")
  if err := http.ListenAndServe(":8080", r); err != nil {
    log.Fatalf("server: %v", err)
  }
}
```

Notes

- `RequestIDMiddleware` adds an `X-Request-ID` header when one is not present and stores the request id in the request's context.
- `LoggerMiddleware(logger)` adds a request-scoped logger into the context which includes request metadata (method, path, request_id, client IP).
- Retrieve the logger with `shttp.LoggerFromContext(ctx)`.
