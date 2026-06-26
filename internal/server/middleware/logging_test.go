package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestLoggerLogsSuccess(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)

	RequestLogger(logger)(handler).ServeHTTP(recorder, request)

	var entry map[string]any
	if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
		t.Fatalf("expected valid JSON log entry: %v", err)
	}

	if entry["method"] != http.MethodGet {
		t.Fatalf("expected method %q; got %v", http.MethodGet, entry["method"])
	}
	if entry["path"] != "/test" {
		t.Fatalf("expected path %q; got %v", "/test", entry["path"])
	}
	if entry["status"] != float64(http.StatusOK) {
		t.Fatalf("expected status %d; got %v", http.StatusOK, entry["status"])
	}
	if _, ok := entry["latency_ms"]; !ok {
		t.Fatal("expected latency_ms field")
	}
	if entry["request_id"] != "unknown" {
		t.Fatalf("expected request_id %q; got %v", "unknown", entry["request_id"])
	}
}

func TestRequestLoggerCapturesStatusCode(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/missing", nil)

	RequestLogger(logger)(handler).ServeHTTP(recorder, request)

	var entry map[string]any
	if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
		t.Fatalf("expected valid JSON log entry: %v", err)
	}

	if entry["status"] != float64(http.StatusNotFound) {
		t.Fatalf("expected status %d; got %v", http.StatusNotFound, entry["status"])
	}
}

func TestRequestLoggerHasLatencyField(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	RequestLogger(logger)(handler).ServeHTTP(recorder, request)

	var entry map[string]any
	if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
		t.Fatalf("expected valid JSON log entry: %v", err)
	}

	latency, ok := entry["latency_ms"].(float64)
	if !ok {
		t.Fatal("expected latency_ms to be a number")
	}
	if latency < 0 {
		t.Fatalf("expected non-negative latency_ms; got %v", latency)
	}
}

func TestRequestLoggerPassesThrough(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/resource", nil)

	RequestLogger(slog.New(slog.NewTextHandler(ioDiscard{}, nil)))(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d; got %d", http.StatusNoContent, recorder.Code)
	}
}
