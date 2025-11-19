package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
)

// recordingHandler captures slog records for assertions.
type recordingHandler struct {
	records *[]slog.Record
	attrs   []slog.Attr
}

func newRecordingHandler() *recordingHandler {
	return &recordingHandler{
		records: &[]slog.Record{},
	}
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	// Copy the record so attributes can be iterated after Handle returns.
	cpy := slog.Record{
		Time:    r.Time,
		Message: r.Message,
		Level:   r.Level,
	}
	for _, attr := range h.attrs {
		cpy.AddAttrs(attr)
	}
	r.Attrs(func(attr slog.Attr) bool {
		cpy.AddAttrs(attr)
		return true
	})
	*h.records = append(*h.records, cpy)
	return nil
}

func (h *recordingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cp := append([]slog.Attr{}, h.attrs...)
	cp = append(cp, attrs...)
	return &recordingHandler{
		records: h.records,
		attrs:   cp,
	}
}

func (h *recordingHandler) WithGroup(string) slog.Handler { return h }

func TestContextLoggerMiddlewareAddsAttrs(t *testing.T) {
	tests := []struct {
		name            string
		remoteAddr      string
		traceID         string
		requestID       string
		wantClientIP    string
		wantTraceIDAttr string
	}{
		{
			name:            "with trace id and request id",
			remoteAddr:      "10.0.0.1:1234",
			traceID:         "trace-abc",
			requestID:       "req-123",
			wantClientIP:    "10.0.0.1",
			wantTraceIDAttr: "trace-abc",
		},
		{
			name:         "fallback trace id uses request id",
			remoteAddr:   "127.0.0.1:9999",
			requestID:    "req-456",
			wantClientIP: "127.0.0.1",
			// trace id should equal request id when header missing
			wantTraceIDAttr: "req-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recHandler := newRecordingHandler()
			logger := slog.New(recHandler)
			handler := contextLoggerMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				log := loggerFromContext(r.Context(), nil)
				log.InfoContext(r.Context(), "test")
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			req.Header.Set("X-Trace-ID", tt.traceID)
			req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDKey, tt.requestID))

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if len(*recHandler.records) == 0 {
				t.Fatalf("expected a log record to be captured")
			}

			record := (*recHandler.records)[0]
			var gotClientIP, gotTraceID string
			record.Attrs(func(attr slog.Attr) bool {
				if attr.Key == "client_ip" {
					gotClientIP = attr.Value.String()
				}
				if attr.Key == "trace_id" {
					gotTraceID = attr.Value.String()
				}
				return true
			})

			if gotClientIP != tt.wantClientIP {
				t.Fatalf("client_ip got %q, want %q", gotClientIP, tt.wantClientIP)
			}
			if gotTraceID != tt.wantTraceIDAttr {
				t.Fatalf("trace_id got %q, want %q", gotTraceID, tt.wantTraceIDAttr)
			}
		})
	}
}

func TestClientIPFromRequest(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		want       string
	}{
		{
			name:       "uses x-forwarded-for first value",
			remoteAddr: "10.0.0.1:1234",
			xff:        "203.0.113.10, 203.0.113.11",
			want:       "203.0.113.10",
		},
		{
			name:       "falls back to remote addr host",
			remoteAddr: "192.168.1.5:8080",
			want:       "192.168.1.5",
		},
		{
			name:       "returns raw remote addr when split fails",
			remoteAddr: "bad-addr",
			want:       "bad-addr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}

			got := clientIPFromRequest(req)
			if got != tt.want {
				t.Fatalf("clientIPFromRequest() = %q, want %q", got, tt.want)
			}
		})
	}
}
