package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoveryCatchesPanicAndReturnsGenericJSON(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("test")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	Recovery(logger)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d; got %d", http.StatusInternalServerError, recorder.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON response body: %v", err)
	}
	if body["error"] != internalServerErrorMessage {
		t.Fatalf("expected generic error %q; got %q", internalServerErrorMessage, body["error"])
	}
	if strings.Contains(recorder.Body.String(), "test") {
		t.Fatal("expected response body not to leak panic value")
	}
	if !strings.Contains(logs.String(), "test") {
		t.Fatal("expected panic value to be logged")
	}
	if !strings.Contains(logs.String(), "stack") {
		t.Fatal("expected stack trace to be logged")
	}
}

func TestRecoveryPassesThroughSuccessfulHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	Recovery(slog.New(slog.NewTextHandler(ioDiscard{}, nil)))(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d; got %d", http.StatusNoContent, recorder.Code)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(data []byte) (int, error) {
	return len(data), nil
}
