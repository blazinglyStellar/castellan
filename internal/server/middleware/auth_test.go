package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/repository/db"

	"castellan/internal/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type mockKeyValidator struct {
	key repository.ApiKey
	err error
}

func (m *mockKeyValidator) ValidateKey(_ context.Context, _ string) (repository.ApiKey, error) {
	return m.key, m.err
}

type authRejectTestCase struct {
	rawKey     string
	mock       *mockKeyValidator
	wantStatus int
	wantError  string
}

func runRejectedTest(t *testing.T, tt authRejectTestCase) {
	t.Helper()
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/gateway/", nil)
	request.Header.Set("Authorization", "Bearer "+tt.rawKey)

	AuthCheck(tt.mock)(handler).ServeHTTP(recorder, request)

	if recorder.Code != tt.wantStatus {
		t.Fatalf("expected status %d; got %d", tt.wantStatus, recorder.Code)
	}
	if called {
		t.Fatal("expected downstream handler not to be called")
	}

	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON response body: %v", err)
	}
	if body["error"] != tt.wantError {
		t.Fatalf("expected error %q; got %q", tt.wantError, body["error"])
	}
}

func TestAuthCheckMissingHeader(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/gateway/", nil)

	mock := &mockKeyValidator{}
	AuthCheck(mock)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d; got %d", http.StatusUnauthorized, recorder.Code)
	}
	if called {
		t.Fatal("expected downstream handler not to be called")
	}

	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON response body: %v", err)
	}
	if body["error"] != "missing authorization header" {
		t.Fatalf("expected error %q; got %q", "missing authorization header", body["error"])
	}
}

func TestAuthCheckMalformedHeader(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{name: "empty authorization", header: ""},
		{name: "wrong scheme", header: "Basic xxx"},
		{name: "wrong prefix", header: "Bearer xyz_xxx"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/gateway/", nil)
			request.Header.Set("Authorization", tt.header)

			mock := &mockKeyValidator{}
			AuthCheck(mock)(handler).ServeHTTP(recorder, request)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("expected status %d; got %d", http.StatusUnauthorized, recorder.Code)
			}
			if called {
				t.Fatal("expected downstream handler not to be called")
			}
		})
	}
}

func TestAuthCheckUnknownKeyHash(t *testing.T) {
	runRejectedTest(t, authRejectTestCase{
		rawKey:     "ca_test-invalid-key-that-does-not-exist",
		mock:       &mockKeyValidator{err: pgx.ErrNoRows},
		wantStatus: http.StatusUnauthorized,
		wantError:  "invalid api key",
	})
}

func TestAuthCheckDBError(t *testing.T) {
	runRejectedTest(t, authRejectTestCase{
		rawKey:     "ca_test-key-with-db-error",
		mock:       &mockKeyValidator{err: context.DeadlineExceeded},
		wantStatus: http.StatusInternalServerError,
		wantError:  "internal error",
	})
}

func TestAuthCheckRevokedKey(t *testing.T) {
	runRejectedTest(t, authRejectTestCase{
		rawKey: "ca_revoked-key-for-testing",
		mock: &mockKeyValidator{
			key: repository.ApiKey{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				KeyHash:   auth.HashKey("ca_revoked-key-for-testing"),
				Status:    repository.ApiKeyStatusRevoked,
				CreatedAt: time.Now().UTC(),
			},
		},
		wantStatus: http.StatusUnauthorized,
		wantError:  "api key revoked",
	})
}

func TestAuthCheckExpiredKey(t *testing.T) {
	runRejectedTest(t, authRejectTestCase{
		rawKey: "ca_expired-key-test",
		mock: &mockKeyValidator{
			key: repository.ApiKey{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				KeyHash:   auth.HashKey("ca_expired-key-test"),
				Status:    repository.ApiKeyStatusActive,
				CreatedAt: time.Now().UTC().Add(-48 * time.Hour),
				ExpiresAt: pgtype.Timestamptz{Time: time.Now().UTC().Add(-1 * time.Hour), Valid: true},
			},
		},
		wantStatus: http.StatusUnauthorized,
		wantError:  "api key expired",
	})
}

func TestAuthCheckValidKey(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)

		consumer := gatewaycontext.GetConsumerInfo(r.Context())
		if !consumer.IsAuthenticated {
			t.Error("expected IsAuthenticated to be true")
		}
		if consumer.ConsumerID == "" {
			t.Error("expected ConsumerID to be set")
		}
		if consumer.APIKeyID == "" {
			t.Error("expected APIKeyID to be set")
		}
	})

	userID := uuid.New()
	keyID := uuid.New()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/gateway/", nil)
	request.Header.Set("Authorization", "Bearer ca_valid-key-for-testing")

	mock := &mockKeyValidator{
		key: repository.ApiKey{
			ID:        keyID,
			UserID:    userID,
			KeyHash:   auth.HashKey("ca_valid-key-for-testing"),
			Status:    repository.ApiKeyStatusActive,
			CreatedAt: time.Now().UTC(),
		},
	}
	AuthCheck(mock)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d; got %d", http.StatusOK, recorder.Code)
	}
	if !called {
		t.Fatal("expected downstream handler to be called")
	}
}

func TestAuthCheckRawKeyNotLogged(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
	defer slog.SetDefault(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rawKey := "ca_super-secret-key-that-must-not-appear-in-logs"
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/gateway/", nil)
	request.Header.Set("Authorization", "Bearer "+rawKey)

	userID := uuid.New()
	mock := &mockKeyValidator{
		key: repository.ApiKey{
			ID:        uuid.New(),
			UserID:    userID,
			KeyHash:   auth.HashKey(rawKey),
			Status:    repository.ApiKeyStatusActive,
			CreatedAt: time.Now().UTC(),
		},
	}
	AuthCheck(mock)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d; got %d", http.StatusOK, recorder.Code)
	}

	logOutput := buf.String()
	t.Logf("log output: %s", logOutput)

	if strings.Contains(logOutput, rawKey) {
		t.Fatal("raw key must not appear in log output")
	}

	if !strings.Contains(logOutput, userID.String()) {
		t.Fatal("expected consumer_id in log output")
	}
}
