# shttp

[![CI Status](https://github.com/andres-vara/shttp/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/andres-vara/shttp/actions/workflows/ci.yml?branch=main) [![Go Report Card](https://goreportcard.com/badge/github.com/andres-vara/shttp)](https://goreportcard.com/report/github.com/andres-vara/shttp) [![PkgGoDev](https://pkg.go.dev/badge/github.com/andres-vara/shttp)](https://pkg.go.dev/github.com/andres-vara/shttp) [![License](https://img.shields.io/github/license/andres-vara/shttp.svg)](LICENSE)

Lightweight HTTP helpers around the standard library `net/http`.

Key features
- Simple router built on `http.ServeMux` with optional path-parameter extraction.
- Middleware helpers (request ID, logging, recovery, CORS, timeout, etc.).
- Tiny surface area so it is easy to embed in small services and tools.

Quick start

Install with Go modules:

```sh
go get github.com/andres-vara/shttp
```

Example: Basic handler with path parameter

```go
package main

import (
"context"
"net/http"
"github.com/andres-vara/shttp"
)

func main() {
	r := shttp.NewRouter()

	r.GET("/hello/{name}", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
name := shttp.PathValue(r, "name")
w.Write([]byte("hello " + name))
return nil
})

	http.ListenAndServe(":8080", r)
}
```

Example: Registering middleware (request-id + structured logger)

```go
package main

import (
"context"
"net/http"
"github.com/andres-vara/shttp"
"github.com/andres-vara/slogr"
)

func main() {
	logger := slogr.New()
	r := shttp.NewRouter()

	// Apply middleware globally on the router
	r.Use(
shttp.RequestIDMiddleware(),
		shttp.LoggerMiddleware(logger),
	)

	r.GET("/users/{id}", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
id := shttp.PathValue(r, "id")
w.Write([]byte("user: " + id))
return nil
})

	http.ListenAndServe(":8080", r)
}
```

Path parameters

Define routes using `{param}` segments (for example `/users/{id}`). The router will extract path parameters and make them available via `shttp.PathValue(r, "id").

## Unified Logging with slogr

shttp integrates seamlessly with the [slogr](https://github.com/andres-vara/slogr) structured logging package. Use `DefaultMiddlewareStack()` to get a ready-to-use middleware stack that includes automatic request ID generation, user context extraction, structured logging with auto-injected request metadata (request_id, user_id, client_ip), request/response logging, and panic recovery.

**Example: Quick start with DefaultMiddlewareStack**

```go
package main

import (
	"context"
	"net/http"
	"github.com/andres-vara/shttp"
	"github.com/andres-vara/slogr"
)

func main() {
	logger := slogr.New(os.Stdout, slogr.DefaultOptions())
	server := shttp.New(context.Background(), nil)
	
	// Use the recommended middleware stack in one line!
	server.Use(shttp.DefaultMiddlewareStack(logger)...)
	
	server.GET("/users/{id}", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		logger := shttp.GetLogger(ctx) // request_id, user_id, client_ip auto-injected
		logger.Info(ctx, "Fetching user", "id", shttp.PathValue(r, "id"))
		w.Write([]byte("user data"))
		return nil
	})
	
	server.Start()
}
```

**Configure logging via Server.Config**

Control logging format (JSON/Text) and level without code changes:

```go
config := &shttp.Config{
	Addr: ":8080",
	LoggerOptions: &slogr.Options{
		Level:       slog.LevelDebug,
		HandlerType: slogr.HandlerTypeJSON, // Switch to HandlerTypeText for human-readable output
		AddLevelPrefix: true,
	},
}
server := shttp.New(ctx, config)
server.Use(shttp.DefaultMiddlewareStack(server.GetLogger())...)
```

Running tests

From the repository root:

```sh
go test ./... -v
```

CI

This repository includes a GitHub Actions workflow (`.github/workflows/ci.yml`) that runs the `slogr` and `shttp` test suites. The badge at the top of this README reflects the latest workflow status.

Examples

- `examples/basic`: minimal server using `shttp.NewRouter()`.
- `examples/middleware`: demonstrates middleware stacking and request-scoped logging.
- `examples/default-middleware`: showcases `DefaultMiddlewareStack()` for quick integration with auto-injected request attributes.
- `examples/config-logger`: demonstrates config-driven logger options (JSON/Text format, level control).

Contributing

Contributions welcome — please open issues or PRs. For local development you can use a `replace` directive in `go.mod` to point to a locally cloned `slogr`:

```go
replace github.com/andres-vara/slogr => ../slogr
```

License

MIT

---
Updated README — 2025-11-13
