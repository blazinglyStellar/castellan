package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxBodySizeRejectsContentLengthOverLimit(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("too large"))
	recorder := httptest.NewRecorder()

	MaxBodySize(3)(handler).ServeHTTP(recorder, request)

	if called {
		t.Fatal("expected handler not to be called")
	}
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d; got %d", http.StatusRequestEntityTooLarge, recorder.Code)
	}
}

func TestMaxBodySizeRejectsStreamingBodyOverLimit(t *testing.T) {
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)

		var maxBytesErr *http.MaxBytesError
		if !errors.As(err, &maxBytesErr) {
			t.Fatalf("expected MaxBytesError; got %v", err)
		}
	})

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("streamed body"))
	request.ContentLength = -1
	recorder := httptest.NewRecorder()

	MaxBodySize(3)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d; got %d", http.StatusRequestEntityTooLarge, recorder.Code)
	}
}

func TestMaxBodySizeAllowsBodyWithinLimit(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("unexpected read error: %v", err)
		}
		if string(body) != "ok" {
			t.Fatalf("expected body %q; got %q", "ok", string(body))
		}

		w.WriteHeader(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("ok"))
	recorder := httptest.NewRecorder()

	MaxBodySize(3)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d; got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestMaxBodySizeFromEnv(t *testing.T) {
	t.Setenv(maxBodySizeEnvKey, "123")

	if got := MaxBodySizeFromEnv(); got != 123 {
		t.Fatalf("expected max body size %d; got %d", 123, got)
	}
}

func TestMaxBodySizeFromEnvDefaultsForInvalidValue(t *testing.T) {
	t.Setenv(maxBodySizeEnvKey, "invalid")

	if got := MaxBodySizeFromEnv(); got != defaultMaxBodySizeBytes {
		t.Fatalf("expected default max body size %d; got %d", defaultMaxBodySizeBytes, got)
	}
}
