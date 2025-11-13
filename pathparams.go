package shttp

import (
	"context"
	"net/http"
	"strings"
)

// pathParamsKey is the context key used to store path parameters.
type pathParamsKey struct{}

// SetPathValues returns a new *http.Request whose context contains the provided params map.
func SetPathValues(r *http.Request, params map[string]string) *http.Request {
	if params == nil {
		return r
	}
	ctx := context.WithValue(r.Context(), pathParamsKey{}, params)
	return r.WithContext(ctx)
}

// SetPathValue sets a single path parameter on the request and returns the updated request.
// It merges with any existing param map.
func SetPathValue(r *http.Request, key, value string) *http.Request {
	var params map[string]string
	if existing, ok := r.Context().Value(pathParamsKey{}).(map[string]string); ok && existing != nil {
		// copy to avoid mutating original map
		params = make(map[string]string, len(existing)+1)
		for k, v := range existing {
			params[k] = v
		}
	} else {
		params = make(map[string]string)
	}
	params[key] = value
	return SetPathValues(r, params)
}

// PathValue retrieves a path parameter value from the request. Returns empty string if not found.
func PathValue(r *http.Request, key string) string {
	if params, ok := r.Context().Value(pathParamsKey{}).(map[string]string); ok && params != nil {
		return params[key]
	}
	return ""
}

// extractPathParams extracts named parameters from a registered pattern and an actual path.
// Example: pattern "/users/{id}" and path "/users/123" -> map[id]="123"
func extractPathParams(pattern, path string) map[string]string {
	// Normalize leading/trailing slashes then split
	pSegs := strings.Split(strings.Trim(pattern, "/"), "/")
	aSegs := strings.Split(strings.Trim(path, "/"), "/")

	if len(pSegs) != len(aSegs) {
		// If lengths differ, we still try to match trailing empty segment cases
		return nil
	}

	params := make(map[string]string)
	for i := 0; i < len(pSegs); i++ {
		ps := pSegs[i]
		if strings.HasPrefix(ps, "{") && strings.HasSuffix(ps, "}") {
			key := strings.TrimSuffix(strings.TrimPrefix(ps, "{"), "}")
			params[key] = aSegs[i]
		}
	}

	return params
}
