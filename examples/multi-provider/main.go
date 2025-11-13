package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andres-vara/shttp"
	"github.com/andres-vara/slogr"
)

func main() {
	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slogr.New(os.Stdout, slogr.DefaultOptions())

	// Create a new server with default configuration
	server := shttp.New(ctx, nil)

	// Middlewares: request ID, contextual logger, request logging, recovery
	server.Use(
		shttp.RequestIDMiddleware(),
		shttp.ContextualLogger(logger),
		shttp.LoggerMiddleware(logger),
		shttp.LoggingMiddleware(logger),
		shttp.RecoveryMiddleware(logger),
	)

	providers := map[string]Provider{
		"prv1": NewProvider1(),
		"prv2": NewProvider2(),
	}
	app := NewApp(providers)
	addRoutes(server, app)

	// Set up a channel to handle shutdown signals
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	go func() {
		log.Println("Starting server at http://localhost:8080")
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-done
	log.Println("Server is shutting down...")

	// Create a deadline for the shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server gracefully stopped")
}

// addRoutes registers provider-agnostic routes. The provider name is a path
// parameter, so handlers can resolve the implementation dynamically.
func addRoutes(server *shttp.Server, app *App) {
	// POST   /providers/{provider}/users
	server.POST("/providers/{provider}/users", app.withProvider(app.handleCreateUser))
	// PUT    /providers/{provider}/users/{id}
	server.PUT("/providers/{provider}/users/{id}", app.withProvider(app.handleUpdateUser))
	// DELETE /providers/{provider}/users/{id}
	server.DELETE("/providers/{provider}/users/{id}", app.withProvider(app.handleDeleteUser))
}

// App holds the provider registry and exposes handlers.
type App struct {
	providers map[string]Provider
}

func NewApp(providers map[string]Provider) *App {
	return &App{providers: providers}
}

// withProvider abstracts provider resolution to avoid repeating handler logic
// across routes. It reads `{provider}` from the path and injects the resolved
// Provider into the given function.
func (a *App) withProvider(fn func(p Provider, ctx context.Context, w http.ResponseWriter, r *http.Request) error) shttp.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := shttp.PathValue(r, "provider")
		p, ok := a.providers[name]
		if !ok {
			return shttp.NewHTTPError(http.StatusNotFound, fmt.Sprintf("unknown provider: %s", name))
		}
		return fn(p, ctx, w, r)
	}
}

// handleCreateUser handles user creation for any provider.
func (a *App) handleCreateUser(p Provider, ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	l := shttp.GetLogger(ctx)
	var in User
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		return shttp.NewHTTPError(http.StatusBadRequest, "invalid JSON body")
	}
	created, err := p.CreateUser(in)
	if err != nil {
		return err
	}
	if l != nil {
		l.Infof(ctx, "created user id=%s name=%s", created.ID, created.Name)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(created)
}

// handleUpdateUser updates a user for any provider.
func (a *App) handleUpdateUser(p Provider, ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	id := shttp.PathValue(r, "id")
	if id == "" {
		return shttp.NewHTTPError(http.StatusBadRequest, "missing id")
	}
	var in User
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		return shttp.NewHTTPError(http.StatusBadRequest, "invalid JSON body")
	}
	if err := p.UpdateUser(id, in); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// handleDeleteUser deletes a user for any provider.
func (a *App) handleDeleteUser(p Provider, ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	id := shttp.PathValue(r, "id")
	if id == "" {
		return shttp.NewHTTPError(http.StatusBadRequest, "missing id")
	}
	if err := p.DeleteUser(id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// User is a minimal user payload for the demo.
type User struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Provider interface {
	CreateUser(user User) (*User, error)
	DeleteUser(id string) error
	UpdateUser(id string, user User) error
}

// Provider1 implement Provider
type Provider1 struct{}

func NewProvider1() *Provider1 { return &Provider1{} }

func (p *Provider1) CreateUser(user User) (*User, error) {
	// Demo implementation: assign an ID with provider prefix
	user.ID = fmt.Sprintf("p1-%d", time.Now().UnixNano())
	return &user, nil
}

func (p *Provider1) DeleteUser(id string) error { return nil }

func (p *Provider1) UpdateUser(id string, user User) error { return nil }

// Provider2 implement Provider
type Provider2 struct{}

func NewProvider2() *Provider2 { return &Provider2{} }

func (p *Provider2) CreateUser(user User) (*User, error) {
	user.ID = fmt.Sprintf("p2-%d", time.Now().UnixNano())
	return &user, nil
}

func (p *Provider2) DeleteUser(id string) error { return nil }

func (p *Provider2) UpdateUser(id string, user User) error { return nil }
