package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/andres-vara/slogr"
)

func main() {
	logger := slogr.New(os.Stdout, slogr.DefaultOptions())

	// Create chi router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(requestIDToSlogr)
	r.Use(slogrContextMiddleware(logger))
	r.Use(AuthMiddleware)
	r.Use(loggingMiddleware)
	r.Use(middleware.Recoverer)

	// Initialize providers
	providers := map[string]Provider{
		"prv1": NewProvider1(),
		"prv2": NewProvider2(),
	}
	app := NewApp(providers)

	// Register routes
	addRoutes(r, app)

	// Set up a channel to handle shutdown signals
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	go func() {
		log.Println("Starting chi multi-provider example at http://localhost:8080")
		log.Println("Try:")
		log.Println("  curl -X POST http://localhost:8080/providers/prv1/users -H 'Content-Type: application/json' -d '{\"name\":\"Alice\",\"email\":\"alice@example.com\"}'")
		log.Println("  curl -X POST http://localhost:8080/providers/prv2/users -H 'Content-Type: application/json' -d '{\"name\":\"Bob\",\"email\":\"bob@example.com\"}'")
		if err := http.ListenAndServe(":8080", r); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-done
	log.Println("Server is shutting down...")

	// Create a deadline for the shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Wait for shutdown (in real app, would stop server here)
	<-shutdownCtx.Done()
	log.Println("Server gracefully stopped")
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
}

func NewApp(providers map[string]Provider) *App {
	return &App{
		providers: providers,
		users:     make(map[string]*User),
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
	log := slogr.FromContext(ctx)
	if log == nil {
		log = slogr.GetDefaultLogger()
	}

	p, err := a.getProvider(w, r)
	if err != nil {
		log.Warn(ctx, "provider_not_found", slog.String("provider", chi.URLParam(r, "provider")))
		return
	}

	var in User
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		log.Warn(ctx, "invalid_request_body", slog.String("error", err.Error()))
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	created, err := p.CreateUser(in)
	if err != nil {
		log.Error(ctx, "failed_to_create_user", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Store in simple in-memory store
	a.users[created.ID] = created

	log.Info(ctx, "user_created",
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
	log := slogr.FromContext(ctx)
	if log == nil {
		log = slogr.GetDefaultLogger()
	}

	_, err := a.getProvider(w, r)
	if err != nil {
		return
	}

	log.Info(ctx, "list_users", slog.String("provider", chi.URLParam(r, "provider")))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a.users)
}

// updateUserHandler updates a user for any provider
func (a *App) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := slogr.FromContext(ctx)
	if log == nil {
		log = slogr.GetDefaultLogger()
	}

	p, err := a.getProvider(w, r)
	if err != nil {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		log.Warn(ctx, "missing_user_id")
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	var in User
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		log.Warn(ctx, "invalid_request_body", slog.String("error", err.Error()))
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if err := p.UpdateUser(id, in); err != nil {
		log.Error(ctx, "failed_to_update_user",
			slog.String("error", err.Error()),
			slog.String("user_id", id))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update in-memory store
	in.ID = id
	a.users[id] = &in

	log.Info(ctx, "user_updated",
		slog.String("user_id", id),
		slog.String("name", in.Name),
		slog.String("provider", chi.URLParam(r, "provider")))

	w.WriteHeader(http.StatusNoContent)
}

// deleteUserHandler deletes a user for any provider
func (a *App) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := slogr.FromContext(ctx)
	if log == nil {
		log = slogr.GetDefaultLogger()
	}

	p, err := a.getProvider(w, r)
	if err != nil {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		log.Warn(ctx, "missing_user_id")
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	if err := p.DeleteUser(id); err != nil {
		log.Error(ctx, "failed_to_delete_user",
			slog.String("error", err.Error()),
			slog.String("user_id", id))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Remove from in-memory store
	delete(a.users, id)

	log.Info(ctx, "user_deleted",
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

// requestIDToSlogr extracts chi's request ID and adds it to context as slog attr
func requestIDToSlogr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		requestID := middleware.GetReqID(ctx)
		if requestID != "" {
			ctx = slogr.WithAttrs(ctx, slog.String("request_id", requestID))
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// slogrContextMiddleware injects the logger into the request context
func slogrContextMiddleware(logger *slogr.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := slogr.WithLogger(r.Context(), logger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// loggingMiddleware logs incoming requests and outgoing responses
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := slogr.FromContext(ctx)
		if log == nil {
			log = slogr.GetDefaultLogger()
		}

		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		log.Info(ctx, "request_start",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path))

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Milliseconds()
		log.Info(ctx, "request_complete",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
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
		log := slogr.FromContext(ctx)
		if log == nil {
			log = slogr.GetDefaultLogger()
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Warn(ctx, "missing_authorization_header")
			http.Error(w, "missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Extract token from "Bearer {token}" format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Warn(ctx, "invalid_authorization_format")
			http.Error(w, "invalid Authorization format", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// For demo: parse token as base64-encoded JSON
		// In production, you would use JWT library and verify signature
		claims, err := parseToken(token)
		if err != nil {
			log.Warn(ctx, "invalid_token",
				slog.String("error", err.Error()))
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// Inject claims into context as attributes
		ctx = slogr.WithAttrs(ctx,
			slog.String("client_ip", claims.ClientIP),
			slog.String("request_id_claim", claims.RequestID))

		log.Info(ctx, "auth_success",
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

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
