# Chi Multi-Provider Example

This example demonstrates a multi-provider architecture using Chi router and slogr structured logging.

## Pattern Overview

The multi-provider pattern allows you to:
- Register multiple backend implementations for the same domain logic
- Route requests to different providers based on a path parameter
- Share handler logic while delegating implementation to specific providers
- Log operations with request-scoped context

## Architecture

### Key Components

**Provider Interface**
```go
type Provider interface {
    CreateUser(user User) (*User, error)
    UpdateUser(id string, user User) error
    DeleteUser(id string) error
}
```

**Implementations**
- `Provider1`: Prefixes user IDs with `p1-`
- `Provider2`: Prefixes user IDs with `p2-`

**Routes**
- `POST /providers/{provider}/users` - Create user via selected provider
- `PUT /providers/{provider}/users/{id}` - Update user
- `DELETE /providers/{provider}/users/{id}` - Delete user
- `GET /providers/{provider}/users` - List users

### Request Flow

```
Request
  ↓
Middleware (request ID, logger injection, logging)
  ↓
Route Handler (e.g., createUserHandler)
  ↓
getProvider() - Resolve {provider} param to implementation
  ↓
p.CreateUser() - Delegate to specific provider
  ↓
Response with structured logging
```

## Running the Example

```bash
go run main.go
```

The server starts on `http://localhost:8080`.

### Automated Testing

Run the included test script to verify the AuthMiddleware:

```bash
chmod +x test-auth.sh
./test-auth.sh
```

This script tests:
- Request without Authorization header (401)
- Request with invalid auth format (401)
- Successful user creation with valid token
- Multiple providers with different auth claims
- List users endpoint with auth

### Example Requests

**Create user via Provider1:**
```bash
curl -X POST http://localhost:8080/providers/prv1/users \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer {"client_ip":"192.168.1.100","request_id":"req-001"}' \
  -d '{"name":"Alice","email":"alice@example.com"}'
```

**Create user via Provider2:**
```bash
curl -X POST http://localhost:8080/providers/prv2/users \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer {"client_ip":"192.168.1.101","request_id":"req-002"}' \
  -d '{"name":"Bob","email":"bob@example.com"}'
```

**List users:**
```bash
curl http://localhost:8080/providers/prv1/users \
  -H 'Authorization: Bearer {"client_ip":"192.168.1.100","request_id":"req-003"}'
```

**Update user:**
```bash
curl -X PUT http://localhost:8080/providers/prv1/users/p1-1234567890 \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer {"client_ip":"192.168.1.100","request_id":"req-004"}' \
  -d '{"name":"Alice Updated","email":"alice.updated@example.com"}'
```

**Delete user:**
```bash
curl -X DELETE http://localhost:8080/providers/prv1/users/p1-1234567890 \
  -H 'Authorization: Bearer {"client_ip":"192.168.1.100","request_id":"req-005"}'
```

### Authentication

The example includes an `AuthMiddleware` that validates incoming requests:

**Authorization Header Format:**
```
Authorization: Bearer {token}
```

**Token Format (Demo):**
For demonstration, the token is a JSON representation containing claims:
```json
{"client_ip":"192.168.1.100","request_id":"req-001"}
```

**Claim Fields:**
- `client_ip`: Client IP address extracted from the token
- `request_id`: Unique request identifier from the token

**Validation Flow:**
1. AuthMiddleware extracts the Authorization header
2. Parses the "Bearer {token}" format
3. Decodes and validates the token claims
4. Injects `client_ip` and `request_id` into the slogr context
5. These claims are automatically included in all subsequent logs

**Demo vs Production:**
In this example, the token is a plain JSON string for simplicity. In production, you should:
- Use a proper JWT library (e.g., `github.com/golang-jwt/jwt`)
- Verify cryptographic signatures
- Handle token expiration
- Validate issuer and audience claims

**Auth Errors:**
- Missing Authorization header → 401 Unauthorized
- Invalid header format → 401 Unauthorized
- Invalid token → 401 Unauthorized

## Logging Integration

### Middleware Stack

1. **RequestID**: Adds unique request identifier
2. **requestIDToSlogr**: Injects request ID into slogr context
3. **slogrContextMiddleware**: Injects logger into request context
4. **AuthMiddleware**: Validates Authorization header token and extracts claims (client_ip, request_id) into context
5. **loggingMiddleware**: Logs request start/end with timing
6. **Recoverer**: Chi's panic recovery middleware

### Log Output

Each request produces structured logs:

```json
{
  "time": "2024-11-14T...",
  "level": "INFO",
  "msg": "auth_success",
  "client_ip": "192.168.1.100"
}

{
  "time": "2024-11-14T...",
  "level": "INFO",
  "msg": "request_start",
  "method": "POST",
  "path": "/providers/prv1/users",
  "request_id": "xyz-123",
  "client_ip": "192.168.1.100",
  "request_id_claim": "req-001"
}

{
  "time": "2024-11-14T...",
  "level": "INFO",
  "msg": "user_created",
  "user_id": "p1-1234567890",
  "name": "Alice",
  "provider": "prv1",
  "request_id": "xyz-123",
  "client_ip": "192.168.1.100",
  "request_id_claim": "req-001"
}

{
  "time": "2024-11-14T...",
  "level": "INFO",
  "msg": "request_complete",
  "method": "POST",
  "path": "/providers/prv1/users",
  "status": 201,
  "duration_ms": 5,
  "request_id": "xyz-123",
  "client_ip": "192.168.1.100",
  "request_id_claim": "req-001"
}
```

## Key Design Decisions

### Shared Handler Logic
Handlers don't know about specific providers - they work with the `Provider` interface:
```go
func (a *App) createUserHandler(w http.ResponseWriter, r *http.Request) {
    p, err := a.getProvider(w, r)  // Resolve provider
    created, err := p.CreateUser(in)  // Delegate to implementation
}
```

### Context-Based Logger Access
Handlers retrieve the logger from context:
```go
log := slogr.FromContext(ctx)
if log == nil {
    log = slogr.GetDefaultLogger()
}
```

### Error Handling
Errors are handled consistently:
- Unknown provider → 404
- Invalid JSON → 400
- Provider errors → 500

### Graceful Shutdown
The example includes signal handling (SIGINT, SIGTERM) for clean shutdown.

## Extension Points

To extend this example:

1. **Add more providers**: Implement `Provider` interface for new backend types
2. **Add database persistence**: Replace in-memory `a.users` map with database queries
3. **Add authentication**: Add auth middleware before `getProvider()`
4. **Add request validation**: Add middleware to validate request bodies
5. **Add request tracing**: Use distributed trace IDs instead of request IDs
6. **Add metrics**: Count operations per provider

## Comparison with shttp Multi-Provider

This chi version uses:
- **Standard `http.Handler` model** instead of shttp's custom handlers
- **Chi middleware pattern** (`func(http.Handler) http.Handler`) instead of shttp's pattern
- **`chi.URLParam()`** instead of shttp's `PathValue()`
- **slogr integration via middleware** instead of shttp's built-in middleware

The business logic and pattern remain the same - provider registry, interface abstraction, and context-based logging.

## See Also

- [Chi Router Docs](https://go-chi.io/)
- [slogr Documentation](../../slogr/README.md)
- [shttp Multi-Provider Example](../multi-provider/)
