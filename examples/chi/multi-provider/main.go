package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
)

func main() {
	// Base logger shared by the whole application. Use httplog schemas so the
	// request logs emitted by the middleware follow ECS-style naming.
	schema := httplog.SchemaECS
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:       slog.LevelInfo,
		ReplaceAttr: schema.ReplaceAttr,
	})).With(
		slog.String("service", "chi-multi-provider"),
		slog.String("env", "local"),
	)
	slog.SetDefault(logger) // libraries without explicit loggers still log consistently

	// Create chi router
	r := chi.NewRouter()

	// Global middleware: request ID first, then httplog to emit access logs,
	// then attach a per-request logger into context for handler use.
	r.Use(middleware.RequestID)
	r.Use(httplog.RequestLogger(logger, &httplog.Options{
		Level:         slog.LevelInfo,
		Schema:        schema,
		RecoverPanics: true,
	}))
	r.Use(contextLoggerMiddleware(logger)) // attach logger with request metadata to context
	r.Use(AuthMiddleware)                  // enrich logger with auth/claims data
	r.Use(loggingMiddleware)               // log request lifecycle with the contextual logger

	// Initialize providers
	providers := map[string]Provider{
		"prv1": NewProvider1(),
		"prv2": NewProvider2(),
	}
	app := NewApp(providers, logger)

	// Register routes
	addRoutes(r, app)

	// Set up a channel to handle shutdown signals
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	go func() {
		logger.Info("Starting chi multi-provider example", slog.String("addr", ":8080"))
		logger.Info("Example requests",
			slog.String("prv1", "curl -X POST http://localhost:8080/providers/prv1/users -H 'Content-Type: application/json' -d '{\"name\":\"Alice\",\"email\":\"alice@example.com\"}'"),
			slog.String("prv2", "curl -X POST http://localhost:8080/providers/prv2/users -H 'Content-Type: application/json' -d '{\"name\":\"Bob\",\"email\":\"bob@example.com\"}'"),
		)

		if err := http.ListenAndServe(":8080", r); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", slog.String("error", err.Error()))
		}
	}()

	// Wait for interrupt signal
	<-done
	logger.Info("Server is shutting down...")

	// Create a deadline for the shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Wait for shutdown (in real app, would stop server here)
	<-shutdownCtx.Done()
	logger.Info("Server gracefully stopped")
}

// addRoutes registers provider-agnostic routes. The provider name is a path
// parameter, so handlers can resolve the implementation dynamically.
func addRoutes(r *chi.Mux, app *App) {
	// POST   /providers/{provider}/users
	r.Post("/providers/{provider}/users", app.createUserHandler)
	// PUT    /providers/{provider}/users/{id}
	r.Put("/providers/{provider}/users/{id}", app.updateUserHandler)
	// DELETE /providers/{provider}/users/{id}
	r.Delete("/providers/{provider}/users/{id}", app.deleteUserHandler)
	// GET    /providers/{provider}/users (for demo)
	r.Get("/providers/{provider}/users", app.listUsersHandler)
}

// App holds the provider registry and exposes handlers.
type App struct {
	providers map[string]Provider
	users     map[string]*User // Simple in-memory store per provider
	logger    *slog.Logger
}

func NewApp(providers map[string]Provider, logger *slog.Logger) *App {
	return &App{
		providers: providers,
		users:     make(map[string]*User),
		logger:    logger.With(slog.String("component", "app")),
	}
}

// getProvider resolves the provider from URL params
func (a *App) getProvider(w http.ResponseWriter, r *http.Request) (Provider, error) {
	providerName := chi.URLParam(r, "provider")
	p, ok := a.providers[providerName]
	if !ok {
		http.Error(w, fmt.Sprintf("unknown provider: %s", providerName), http.StatusNotFound)
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
	return p, nil
}

// createUserHandler handles user creation for any provider
func (a *App) createUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := loggerFromContext(ctx, a.logger)

	p, err := a.getProvider(w, r)
	if err != nil {
		log.WarnContext(ctx, "provider_not_found", slog.String("provider", chi.URLParam(r, "provider")))
		return
	}

	var in User
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		log.WarnContext(ctx, "invalid_request_body", slog.String("error", err.Error()))
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	created, err := p.CreateUser(in)
	if err != nil {
		log.ErrorContext(ctx, "failed_to_create_user", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Store in simple in-memory store
	a.users[created.ID] = created

	httplog.SetAttrs(ctx,
		slog.String("user_id", created.ID),
		slog.String("provider", chi.URLParam(r, "provider")))

	log.InfoContext(ctx, "user_created",
		slog.String("user_id", created.ID),
		slog.String("name", created.Name),
		slog.String("provider", chi.URLParam(r, "provider")))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

// listUsersHandler lists users (demo endpoint)
func (a *App) listUsersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := loggerFromContext(ctx, a.logger)

	_, err := a.getProvider(w, r)
	if err != nil {
		return
	}

	httplog.SetAttrs(ctx, slog.String("provider", chi.URLParam(r, "provider")))
	log.InfoContext(ctx, "list_users", slog.String("provider", chi.URLParam(r, "provider")))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a.users)
}

// updateUserHandler updates a user for any provider
func (a *App) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := loggerFromContext(ctx, a.logger)

	p, err := a.getProvider(w, r)
	if err != nil {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		log.WarnContext(ctx, "missing_user_id")
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	var in User
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		log.WarnContext(ctx, "invalid_request_body", slog.String("error", err.Error()))
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if err := p.UpdateUser(id, in); err != nil {
		log.ErrorContext(ctx, "failed_to_update_user",
			slog.String("error", err.Error()),
			slog.String("user_id", id))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update in-memory store
	in.ID = id
	a.users[id] = &in

	httplog.SetAttrs(ctx,
		slog.String("user_id", id),
		slog.String("provider", chi.URLParam(r, "provider")))

	log.InfoContext(ctx, "user_updated",
		slog.String("user_id", id),
		slog.String("name", in.Name),
		slog.String("provider", chi.URLParam(r, "provider")))

	w.WriteHeader(http.StatusNoContent)
}

// deleteUserHandler deletes a user for any provider
func (a *App) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := loggerFromContext(ctx, a.logger)

	p, err := a.getProvider(w, r)
	if err != nil {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		log.WarnContext(ctx, "missing_user_id")
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	if err := p.DeleteUser(id); err != nil {
		log.ErrorContext(ctx, "failed_to_delete_user",
			slog.String("error", err.Error()),
			slog.String("user_id", id))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Remove from in-memory store
	delete(a.users, id)

	httplog.SetAttrs(ctx,
		slog.String("user_id", id),
		slog.String("provider", chi.URLParam(r, "provider")))
	log.InfoContext(ctx, "user_deleted",
		slog.String("user_id", id),
		slog.String("provider", chi.URLParam(r, "provider")))

	w.WriteHeader(http.StatusNoContent)
}

// User is a minimal user payload for the demo
type User struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Provider interface abstracts different user providers
type Provider interface {
	CreateUser(user User) (*User, error)
	UpdateUser(id string, user User) error
	DeleteUser(id string) error
}

// Provider1 implementation
type Provider1 struct{}

func NewProvider1() *Provider1 { return &Provider1{} }

func (p *Provider1) CreateUser(user User) (*User, error) {
	// Demo implementation: assign an ID with provider prefix
	user.ID = fmt.Sprintf("p1-%d", time.Now().UnixNano())
	return &user, nil
}

func (p *Provider1) UpdateUser(id string, user User) error {
	return nil
}

func (p *Provider1) DeleteUser(id string) error {
	return nil
}

// Provider2 implementation
type Provider2 struct{}

func NewProvider2() *Provider2 { return &Provider2{} }

func (p *Provider2) CreateUser(user User) (*User, error) {
	user.ID = fmt.Sprintf("p2-%d", time.Now().UnixNano())
	return &user, nil
}

func (p *Provider2) UpdateUser(id string, user User) error {
	return nil
}

func (p *Provider2) DeleteUser(id string) error {
	return nil
}

// Middleware helpers

// contextLoggerMiddleware builds a per-request logger with context-aware
// metadata (request ID, trace ID, client IP) and stores it on the context so
// handlers can call loggerFromContext(ctx).
func contextLoggerMiddleware(baseLogger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			requestID := middleware.GetReqID(ctx)
			traceID := r.Header.Get("X-Trace-ID")
			if traceID == "" {
				traceID = requestID
			}

			attrs := []slog.Attr{
				slog.String("client_ip", clientIPFromRequest(r)),
			}
			if requestID != "" {
				attrs = append(attrs, slog.String("request_id", requestID))
			}
			if traceID != "" {
				attrs = append(attrs, slog.String("trace_id", traceID))
			}

			// Ensure access logs emitted by httplog carry the enriched attrs.
			httplog.SetAttrs(ctx, attrs...)

			requestLogger := baseLogger.With(attrsToArgs(attrs)...)
			ctx = withLogger(ctx, requestLogger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// loggingMiddleware logs incoming requests and outgoing responses using the
// logger already stored in the context.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := loggerFromContext(ctx, nil)

		requestLog := log.With(
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)

		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		requestLog.InfoContext(ctx, "request_start")

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Milliseconds()
		requestLog.InfoContext(ctx, "request_complete",
			slog.Int("status", wrapped.statusCode),
			slog.Int64("duration_ms", duration))
	})
}

// TokenClaims represents the claims extracted from the authorization token
type TokenClaims struct {
	ClientIP  string `json:"client_ip"`
	RequestID string `json:"request_id"`
}

// AuthMiddleware validates the Authorization header token and extracts claims
// Expected header format: "Bearer {token}"
// For demo purposes, we parse a simple JSON token format
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := loggerFromContext(ctx, nil)

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.WarnContext(ctx, "missing_authorization_header")
			http.Error(w, "missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Extract token from "Bearer {token}" format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.WarnContext(ctx, "invalid_authorization_format")
			http.Error(w, "invalid Authorization format", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// For demo: parse token as base64-encoded JSON
		// In production, you would use JWT library and verify signature
		claims, err := parseToken(token)
		if err != nil {
			log.WarnContext(ctx, "invalid_token",
				slog.String("error", err.Error()))
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// Enrich logger with claims and attach back to the context
		log = log.With(
			slog.String("client_ip", claims.ClientIP),
			slog.String("request_id_claim", claims.RequestID),
		)
		httplog.SetAttrs(ctx,
			slog.String("client_ip", claims.ClientIP),
			slog.String("request_id_claim", claims.RequestID),
		)
		ctx = withLogger(ctx, log)

		log.InfoContext(ctx, "auth_success",
			slog.String("client_ip", claims.ClientIP))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// parseToken decodes the token (demo implementation)
// In production, use a proper JWT library like github.com/golang-jwt/jwt
func parseToken(token string) (*TokenClaims, error) {
	// Demo: treat token as base64-encoded JSON representation
	// For example: token = base64("{\"client_ip\":\"192.168.1.1\",\"request_id\":\"req-123\"}")
	// In production, this would be JWT verification with signature validation

	// For demo purposes, we'll accept a simple JSON token format
	// Expected format: {"client_ip":"192.168.1.1","request_id":"req-123"}
	var claims TokenClaims
	if err := json.Unmarshal([]byte(token), &claims); err != nil {
		return nil, fmt.Errorf("failed to parse token claims: %w", err)
	}

	if claims.ClientIP == "" || claims.RequestID == "" {
		return nil, fmt.Errorf("missing required claim fields")
	}

	return &claims, nil
}

func clientIPFromRequest(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
}

type loggerContextKey struct{}

// withLogger stores a slog logger on the context for handlers/middleware.
func withLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// loggerFromContext recovers a slog logger from context, falling back to the
// provided default or slog.Default() when missing.
func loggerFromContext(ctx context.Context, fallback *slog.Logger) *slog.Logger {
	if ctxLogger, ok := ctx.Value(loggerContextKey{}).(*slog.Logger); ok && ctxLogger != nil {
		return ctxLogger
	}
	if fallback != nil {
		return fallback
	}
	return slog.Default()
}

// attrsToArgs converts slog.Attr slice to variadic args for slog.With.
func attrsToArgs(attrs []slog.Attr) []any {
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	return args
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
