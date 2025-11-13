package shttp

import (
    "context"
    "io"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/andres-vara/slogr"
)

// TestIntegration_ServerWithMiddleware starts an httptest.Server using the
// package router and exercises middleware + path param extraction end-to-end.
func TestIntegration_ServerWithMiddleware(t *testing.T) {
    ctx := context.Background()

    logger := slogr.New(io.Discard, slogr.DefaultOptions())
    cfg := &Config{Addr: ":0", Logger: logger}
    srv := New(ctx, cfg)

    // Register middleware in the expected order
    srv.Use(
        RequestIDMiddleware(),
        ContextualLogger(logger),
        LoggerMiddleware(logger),
        LoggingMiddleware(logger),
        RecoveryMiddleware(logger),
    )

    // Simple handler that returns the path parameter value
    srv.GET("/hello/{name}", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
        name := PathValue(r, "name")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(name))
        return nil
    })

    ts := httptest.NewServer(srv.Router())
    defer ts.Close()

    res, err := http.Get(ts.URL + "/hello/alice")
    if err != nil {
        t.Fatalf("request failed: %v", err)
    }
    defer res.Body.Close()

    if res.StatusCode != http.StatusOK {
        t.Fatalf("expected status 200, got %d", res.StatusCode)
    }

    body, _ := io.ReadAll(res.Body)
    if string(body) != "alice" {
        t.Fatalf("unexpected body: %q", string(body))
    }

    if res.Header.Get("X-Request-ID") == "" {
        t.Fatalf("expected X-Request-ID header set by RequestIDMiddleware")
    }
}
