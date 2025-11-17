module chi-multi-provider

go 1.24.0

require (
	github.com/andres-vara/slogr v0.0.3
	github.com/go-chi/chi/v5 v5.0.0
)

// Use local copy of slogr during development
replace github.com/andres-vara/slogr => ../../../../slogr
