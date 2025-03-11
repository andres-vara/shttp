# shttp Examples

This directory contains example applications demonstrating various features of the `shttp` package.

## Running the Examples

All examples can be run with `go run main.go` from their respective directories.

## Example Applications

### Quick Start

Location: [quickstart/main.go](./quickstart/main.go)

The simplest possible example:
- Minimal HTTP server setup
- Single route handler
- No graceful shutdown

To run:
```bash
cd quickstart
go run main.go
```

Then visit http://localhost:8080/hello in your browser or use curl:
```bash
curl http://localhost:8080/hello
```

### Basic Server

Location: [basic/main.go](./basic/main.go)

A simple HTTP server demonstrating:
- Basic server setup
- Route registration
- Graceful shutdown

To run:
```bash
cd basic
go run main.go
```

Then visit http://localhost:8080 in your browser or use curl:
```bash
curl http://localhost:8080
curl http://localhost:8080/health
```

### Middleware Usage

Location: [middleware/main.go](./middleware/main.go)

Demonstrates various middleware features:
- Request ID generation
- Panic recovery
- Request/response logging
- CORS handling
- Request timeouts
- Custom middleware implementation

To run:
```bash
cd middleware
go run main.go
```

Test different middleware features:
```bash
# Basic request with request ID
curl -v http://localhost:8080

# Slow response (3s but completes before the 5s timeout)
curl http://localhost:8080/slow

# Panic recovery (will return a 500 error but server keeps running)
curl http://localhost:8080/panic

# CORS preflight request
curl -v -X OPTIONS -H "Origin: http://example.com" \
     -H "Access-Control-Request-Method: GET" \
     http://localhost:8080
```

### Error Handling

Location: [error-handling/main.go](./error-handling/main.go)

Demonstrates centralized error handling:
- Custom error types
- Error mapping to HTTP status codes
- JSON error responses

To run:
```bash
cd error-handling
go run main.go
```

Test different error types:
```bash
# Success response
curl http://localhost:8080/success

# 404 Not Found error
curl http://localhost:8080/not-found

# 400 Bad Request error
curl http://localhost:8080/validation-error

# 401 Unauthorized error
curl http://localhost:8080/unauthorized

# 500 Internal Server error
curl http://localhost:8080/server-error
```

### TLS Server

Location: [tls/main.go](./tls/main.go)

Demonstrates HTTPS server setup:
- Self-signed certificate generation
- TLS configuration
- HTTPS server startup

To run:
```bash
cd tls
go run main.go
```

Test the TLS server:
```bash
# Note: -k flag is needed to accept the self-signed certificate
curl -k https://localhost:8443
```

You can also visit https://localhost:8443 in your browser 
(you'll need to accept the security warning for the self-signed certificate). 