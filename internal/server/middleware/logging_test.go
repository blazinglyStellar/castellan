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

func TestRequestLoggerRedactsAuthorizationHeader(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rawKey := "ca_super-secret-api-key-must-not-appear-in-logs"
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/gateway/", nil)
	request.Header.Set("Authorization", "Bearer "+rawKey)

	RequestLogger(logger)(handler).ServeHTTP(recorder, request)

	logOutput := buf.String()

	if strings.Contains(logOutput, rawKey) {
		t.Fatal("raw api key must not appear in request log output")
	}
	if strings.Contains(logOutput, "Bearer "+rawKey) {
		t.Fatal("raw authorization header value must not appear in request log output")
	}

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("expected valid JSON log entry: %v", err)
	}
	if entry["authorization"] != "[REDACTED]" {
		t.Fatalf("expected authorization field to be \"[REDACTED]\"; got %v", entry["authorization"])
	}
}
