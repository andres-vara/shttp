// pathparams.go
package shttp

import (
	"context"
	"net/http"
)

type pathParamsKey struct{}

func setPathParams(r *http.Request, params map[string]string) *http.Request {
	ctx := context.WithValue(r.Context(), pathParamsKey{}, params)
	return r.WithContext(ctx)
}

func GetPathParam(r *http.Request, key string) string {
	if params, ok := r.Context().Value(pathParamsKey{}).(map[string]string); ok {
		return params[key]
	}
	return ""
}


// server.go (or the file where the router is implemented)
package shttp

import (
	"strings"
)

type route struct {
	method   string
	pattern  string
	segments []segment
	handler  Handler
}

type segment struct {
	isParam bool
	value   string
}

func parsePattern(pattern string) []segment {
	parts := strings.Split(strings.Trim(pattern, "/"), "/")
	segments := make([]segment, len(parts))
	for i, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			segments[i] = segment{isParam: true, value: part[1 : len(part)-1]}
		} else {
			segments[i] = segment{isParam: false, value: part}
		}
	}
	return segments
}

func matchRoute(path string, segments []segment) (bool, map[string]string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != len(segments) {
		return false, nil
	}
	params := make(map[string]string)
	for i, part := range parts {
		seg := segments[i]
		if seg.isParam {
			params[seg.value] = part
		} else if seg.value != part {
			return false, nil
		}
	}
	return true, params
}

// In your server's main request handler loop (e.g. ServeHTTP), example usage:
//
// func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//     for _, route := range s.routes {
//         if route.method != r.Method {
//             continue
//         }
//         matched, params := matchRoute(r.URL.Path, route.segments)
//         if matched {
//             r = setPathParams(r, params)
//             err := route.handler(r.Context(), w, r)
//             if err != nil {
//                 // handle error
//             }
//             return
//         }
//     }
//     http.NotFound(w, r)
// }


// pathparams_test.go
package shttp

import (
	"net/http/httptest"
	"testing"
)

func TestGetPathParam(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/123", nil)
	r = setPathParams(r, map[string]string{"id": "123"})

	id := GetPathParam(r, "id")
	if id != "123" {
		t.Errorf("expected id to be '123', got '%s'", id)
	}
}

func TestMatchRoute(t *testing.T) {
	pattern := "/api/{id}"
	segments := parsePattern(pattern)

	matched, params := matchRoute("/api/456", segments)
	if !matched {
		t.Fatalf("route should have matched")
	}

	if params["id"] != "456" {
		t.Errorf("expected param 'id' = '456', got '%s'", params["id"])
	}
}


# improvement recomendations


   1. API Consistency: Make Router.Use variadic like Server.Use.
   2. Code Consolidation: Consider a single Handle method in the router.
   3. Logging: Make the context-aware logging more explicit.
   4. Configuration: Explore the functional options pattern for server configuration.
   5. Documentation: Add more examples and package-level comments.

