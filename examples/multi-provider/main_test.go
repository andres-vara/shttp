package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/andres-vara/shttp"
)

// stubProvider lets tests control Provider behavior.
type stubProvider struct {
    createFn func(User) (*User, error)
    deleteFn func(string) error
    updateFn func(string, User) error
}

func (s *stubProvider) CreateUser(u User) (*User, error)   { return s.createFn(u) }
func (s *stubProvider) DeleteUser(id string) error          { return s.deleteFn(id) }
func (s *stubProvider) UpdateUser(id string, u User) error  { return s.updateFn(id, u) }

func TestHandleCreateUser(t *testing.T) {
    okProvider := &stubProvider{
        createFn: func(u User) (*User, error) {
            u.ID = "pX-1"
            return &u, nil
        },
        deleteFn: func(string) error { return nil },
        updateFn: func(string, User) error { return nil },
    }

    app := NewApp(map[string]Provider{"ok": okProvider})

    tests := []struct {
        name           string
        provider       string
        body           any
        expectStatus   int
        expectErrCode  int // if >0, expect HTTPError with that status
    }{
        {name: "success", provider: "ok", body: User{Name: "Ada", Email: "a@x"}, expectStatus: http.StatusCreated},
        {name: "unknown provider", provider: "missing", body: User{Name: "Ada"}, expectErrCode: http.StatusNotFound},
        {name: "invalid json", provider: "ok", body: "{not-json}", expectErrCode: http.StatusBadRequest},
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            var bodyBytes []byte
            switch v := tc.body.(type) {
            case string:
                bodyBytes = []byte(v)
            default:
                b, _ := json.Marshal(v)
                bodyBytes = b
            }

            req := httptest.NewRequest(http.MethodPost, "/providers/"+tc.provider+"/users", bytes.NewReader(bodyBytes))
            w := httptest.NewRecorder()

            handler := app.withProvider(app.handleCreateUser)
            err := handler(req.Context(), w, req)

            if tc.expectErrCode > 0 {
                if err == nil {
                    t.Fatalf("expected error %d, got nil", tc.expectErrCode)
                }
                if httpErr, ok := err.(shttp.HTTPError); !ok || httpErr.StatusCode != tc.expectErrCode {
                    t.Fatalf("expected HTTPError(%d), got %#v", tc.expectErrCode, err)
                }
                return
            }

            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if w.Code != tc.expectStatus {
                t.Fatalf("status %d, want %d", w.Code, tc.expectStatus)
            }
            // Basic JSON shape check
            var got User
            if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
                t.Fatalf("invalid json: %v", err)
            }
            if got.ID == "" || got.Name == "" {
                t.Fatalf("unexpected body: %+v", got)
            }
        })
    }
}

func TestHandleUpdateUser(t *testing.T) {
    called := false
    okProvider := &stubProvider{
        createFn: func(u User) (*User, error) { return &u, nil },
        deleteFn: func(string) error { return nil },
        updateFn: func(id string, u User) error { called = true; return nil },
    }
    app := NewApp(map[string]Provider{"ok": okProvider})

    t.Run("success", func(t *testing.T) {
        body, _ := json.Marshal(User{Name: "Ada"})
        req := httptest.NewRequest(http.MethodPut, "/providers/ok/users/123", bytes.NewReader(body))
        req.SetPathValue("id", "123")
        w := httptest.NewRecorder()

        handler := app.withProvider(app.handleUpdateUser)
        if err := handler(req.Context(), w, req); err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if w.Code != http.StatusNoContent {
            t.Fatalf("status %d, want %d", w.Code, http.StatusNoContent)
        }
        if !called {
            t.Fatalf("provider.UpdateUser was not called")
        }
    })

    t.Run("missing id", func(t *testing.T) {
        body, _ := json.Marshal(User{Name: "Ada"})
        req := httptest.NewRequest(http.MethodPut, "/providers/ok/users/", bytes.NewReader(body))
        w := httptest.NewRecorder()
        handler := app.withProvider(app.handleUpdateUser)
        err := handler(req.Context(), w, req)
        if err == nil {
            t.Fatalf("expected error, got nil")
        }
        if httpErr, ok := err.(shttp.HTTPError); !ok || httpErr.StatusCode != http.StatusBadRequest {
            t.Fatalf("expected HTTPError(400), got %#v", err)
        }
    })
}

func TestHandleDeleteUser(t *testing.T) {
    called := false
    okProvider := &stubProvider{
        createFn: func(u User) (*User, error) { return &u, nil },
        deleteFn: func(id string) error { called = true; return nil },
        updateFn: func(string, User) error { return nil },
    }
    app := NewApp(map[string]Provider{"ok": okProvider})

    t.Run("success", func(t *testing.T) {
        req := httptest.NewRequest(http.MethodDelete, "/providers/ok/users/abc", nil)
        req.SetPathValue("id", "abc")
        w := httptest.NewRecorder()

        handler := app.withProvider(app.handleDeleteUser)
        if err := handler(req.Context(), w, req); err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if w.Code != http.StatusNoContent {
            t.Fatalf("status %d, want %d", w.Code, http.StatusNoContent)
        }
        if !called {
            t.Fatalf("provider.DeleteUser was not called")
        }
    })

    t.Run("unknown provider", func(t *testing.T) {
        req := httptest.NewRequest(http.MethodDelete, "/providers/missing/users/abc", nil)
        req.SetPathValue("id", "abc")
        w := httptest.NewRecorder()

        handler := app.withProvider(app.handleDeleteUser)
        err := handler(req.Context(), w, req)
        if err == nil {
            t.Fatalf("expected error, got nil")
        }
        if httpErr, ok := err.(shttp.HTTPError); !ok || httpErr.StatusCode != http.StatusNotFound {
            t.Fatalf("expected HTTPError(404), got %#v", err)
        }
    })
}

