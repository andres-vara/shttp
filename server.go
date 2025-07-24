package shttp

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/andres-vara/slogr"
)

// Server represents an HTTP server with structured configuration
type Server struct {
	// The underlying http.Server
	server *http.Server

	// Server configuration
	config *Config

	// Router for handling requests
	router *Router

	// Logger instance
	logger *slogr.Logger

	ctx context.Context
}

// Config holds the server configuration
type Config struct {
	// Address to listen on (e.g., ":8080")
	Addr string

	// Read timeout for the server
	ReadTimeout time.Duration

	// Write timeout for the server
	WriteTimeout time.Duration

	// Idle timeout for the server
	IdleTimeout time.Duration

	// Maximum header size in bytes
	MaxHeaderBytes int

	// Logger instance to use
	Logger *slogr.Logger
}

// DefaultConfig returns a default server configuration
func DefaultConfig() *Config {
	return &Config{
		Addr:           ":8080",
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
		Logger:         slogr.New(os.Stdout, slogr.DefaultOptions()),
	}
}

// New creates a new HTTP server with the given configuration
func New(ctx context.Context, config *Config) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	// Create router
	router := NewRouter()

	// Create server
	server := &http.Server{
		Addr:           config.Addr,
		Handler:        router,
		ReadTimeout:    config.ReadTimeout,
		WriteTimeout:   config.WriteTimeout,
		IdleTimeout:    config.IdleTimeout,
		MaxHeaderBytes: config.MaxHeaderBytes,
	}

	return &Server{
		server: server,
		config: config,
		router: router,
		logger: config.Logger,
		ctx:    ctx,
	}
}

// Start starts the server and begins listening for requests
func (s *Server) Start() error {
	s.logger.Infof(s.ctx, "[server.start] Starting server on %s", s.config.Addr)
	return s.server.ListenAndServe()
}

// StartTLS starts the server with TLS support
func (s *Server) StartTLS(certFile, keyFile string) error {
	s.logger.Infof(s.ctx, "[server.start] Starting TLS server on %s", s.config.Addr)
	return s.server.ListenAndServeTLS(certFile, keyFile)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Infof(s.ctx, "[server.shutdown] Shutting down server")
	return s.server.Shutdown(ctx)
}

// Router returns the server's router
func (s *Server) Router() *Router {
	return s.router
}

// GET registers a GET route handler
func (s *Server) GET(path string, handler Handler) {
	s.router.GET(path, handler)
}

// POST registers a POST route handler
func (s *Server) POST(path string, handler Handler) {
	s.router.POST(path, handler)
}

// PUT registers a PUT route handler
func (s *Server) PUT(path string, handler Handler) {
	s.router.PUT(path, handler)
}

// DELETE registers a DELETE route handler
func (s *Server) DELETE(path string, handler Handler) {
	s.router.DELETE(path, handler)
}

// PATCH registers a PATCH route handler
func (s *Server) PATCH(path string, handler Handler) {
	s.router.PATCH(path, handler)
}

// Handle registers a handler for the given method and path
func (s *Server) Handle(method, path string, handler Handler) {
	s.router.Handle(method, path, handler)
}

// Use adds one or more middleware to the server (variadic approach)
func (s *Server) Use(middleware ...Middleware) {
	s.router.Use(middleware...)
}
