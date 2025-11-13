// pathparams.go
package shttp

import (
	"context"
	"net/http"
)

type pathParamsKey struct{}

func setPathParams(r *http.Request, params map[string]string) *http.Request {
	ctx := context.WithValue(r.Context(), pathParamsKey{}, params)
	return r.WithContext(ctx)
}

func GetPathParam(r *http.Request, key string) string {
	if params, ok := r.Context().Value(pathParamsKey{}).(map[string]string); ok {
		return params[key]
	# shttp

	Lightweight HTTP helpers around the standard library `net/http`.

	Features
	- Simple router built on `http.ServeMux` with optional path-parameter extraction.
	- Middleware helpers (request ID, logging, recovery, CORS, timeout, etc.).
	- Integration-friendly: small surface area and no external router dependencies.

	Quick start
	1. Install (use Go modules):

	```sh
	go get github.com/andres-vara/shttp
	```

	2. Create a server and register handlers:

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

	Path parameters
	- Define routes with `{param}` segments (for example `/users/{id}`). The router will extract path parameters and make them available via `shttp.PathValue(r, "id")`.

	Middleware
	- Use the provided middleware helpers to add structured logging, request IDs, and other cross-cutting concerns.

	Running tests

	From the repository root:

	```sh
	go test ./... -v
	```

	Contributing
	- Open issues or PRs. For local development you can use a `replace` directive in `go.mod` to point to a local `slogr` clone if needed.

	License
	- MIT

	---
	Updated README (docs branch) — 2025-11-13
//         }

//         matched, params := matchRoute(r.URL.Path, route.segments)
