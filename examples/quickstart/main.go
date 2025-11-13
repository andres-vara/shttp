package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/andres-vara/shttp"
)

func main() {
	// Create a context
	ctx := context.Background()

	// Create a new server with default configuration
	server := shttp.New(ctx, nil)

	// Add a simple route
	server.GET("/hello", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		fmt.Fprintln(w, "Hello, World!")
		return nil
	})

	// Start the server
	log.Println("Starting server at http://localhost:8080")
	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
