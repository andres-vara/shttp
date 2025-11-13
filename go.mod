module github.com/andres-vara/shttp

go 1.24.0

require github.com/andres-vara/slogr v0.0.3

// Use local copy of slogr during development
replace github.com/andres-vara/slogr => ../slogr
