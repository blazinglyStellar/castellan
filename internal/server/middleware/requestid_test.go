package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDGeneratesUUID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	RequestID()(handler).ServeHTTP(recorder, request)

	requestID := recorder.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Fatal("expected non-empty X-Request-ID header")
	}
	if len(requestID) != 36 {
		t.Fatalf("expected UUID v4 (36 chars); got %q (%d chars)", requestID, len(requestID))
	}
}

func TestRequestIDPropagatesHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", "my-custom-trace-123")

	RequestID()(handler).ServeHTTP(recorder, request)

	if got := recorder.Header().Get("X-Request-ID"); got != "my-custom-trace-123" {
		t.Fatalf("expected X-Request-ID %q; got %q", "my-custom-trace-123", got)
	}
}

func TestRequestIDSetsResponseHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	RequestID()(handler).ServeHTTP(recorder, request)

	if got := recorder.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("expected non-empty X-Request-ID response header")
	}
}

func TestGetRequestID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r.Context())
		if requestID == "" {
			t.Error("expected non-empty request ID from GetRequestID")
		}
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	RequestID()(handler).ServeHTTP(recorder, request)
}

func TestGetRequestIDReturnsEmptyWithoutMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := GetRequestID(r.Context()); got != "" {
			t.Fatalf("expected empty request ID without middleware; got %q", got)
		}
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(recorder, request)
}
