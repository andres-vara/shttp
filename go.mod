module github.com/andres-vara/shttp

go 1.24.0

require github.com/andres-vara/slogr v0.0.3

require (
	github.com/go-chi/chi/v5 v5.1.0 // indirect
	github.com/go-chi/httplog/v3 v3.3.0 // indirect
)

// Use local copy of slogr during development
replace github.com/andres-vara/slogr => ../slogr
