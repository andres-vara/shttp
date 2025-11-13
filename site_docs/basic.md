# Basic Example — shttp

This document expands the `examples/basic` minimal server and includes some notes on running and testing locally.

## Example server

```go
package main

import (
  "context"
  "log"
  "net/http"
  "os"

  "github.com/andres-vara/shttp"
)

func main() {
  addr := ":8080"
  if v := os.Getenv("PORT"); v != "" {
    addr = ":" + v
  }

  r := shttp.NewRouter()

  // Simple handler that returns a greeting using a path parameter.
  r.GET("/hello/{name}", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    name := shttp.PathValue(r, "name")
    if name == "" {
      name = "world"
    }
    _, _ = w.Write([]byte("Hello " + name + "\n"))
    return nil
  })

  // Start server
  log.Printf("listening on %s", addr)
  if err := http.ListenAndServe(addr, r); err != nil {
    log.Fatalf("server exited: %v", err)
  }
}
```

## Notes

- The router supports path parameters using `{name}` syntax in route patterns.
- Use `shttp.PathValue(r, "param")` to access parameters in handlers.
- For local development you can run `go run ./examples/basic` or build and run the binary.

## Running

```sh
go run ./examples/basic
# or
go build -o hello ./examples/basic && ./hello
```
