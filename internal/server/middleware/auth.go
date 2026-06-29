package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/repository/db"

	"castellan/internal/auth"

	"github.com/jackc/pgx/v5"
)

const bearerPrefix = "Bearer "

// KeyValidator looks up an API key by its SHA-256 hash.
type KeyValidator interface {
	ValidateKey(ctx context.Context, keyHash string) (repository.ApiKey, error)
}

// KeyValidatorFunc is an adapter that lets a function serve as a KeyValidator.
type KeyValidatorFunc func(ctx context.Context, keyHash string) (repository.ApiKey, error)

// ValidateKey delegates to the underlying function.
func (f KeyValidatorFunc) ValidateKey(ctx context.Context, keyHash string) (repository.ApiKey, error) {
	return f(ctx, keyHash)
}

// AuthCheck validates the Authorization: Bearer ca_xxx header on every request.
// It extracts the key, hashes it, looks up the hash in PostgreSQL, checks
// status and expiry, and injects ConsumerInfo into the context on success.
// The raw key is never logged — only consumer_id and key_hash prefix appear.
func AuthCheck(validator KeyValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			if !strings.HasPrefix(authHeader, bearerPrefix) {
				writeJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{errKey: "missing authorization header"})
				return
			}

			rawKey := strings.TrimPrefix(authHeader, bearerPrefix)

			if !strings.HasPrefix(rawKey, "ca_") {
				writeJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{errKey: "invalid api key"})
				return
			}

			keyHash := auth.HashKey(rawKey)

			key, err := validator.ValidateKey(r.Context(), keyHash)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					writeJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{errKey: "invalid api key"})
					return
				}
				slog.ErrorContext(
					r.Context(), "key validation failed",
					slog.String("error", err.Error()),
				)
				writeJSON(r.Context(), w, http.StatusInternalServerError, map[string]string{errKey: "internal error"})
				return
			}

			if key.Status == repository.ApiKeyStatusRevoked {
				writeJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{errKey: "api key revoked"})
				return
			}

			if key.ExpiresAt.Valid && key.ExpiresAt.Time.Before(time.Now()) {
				writeJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{errKey: "api key expired"})
				return
			}

			slog.DebugContext(
				r.Context(), "authenticated request",
				slog.String("consumer_id", key.UserID.String()),
				slog.String("key_hash_prefix", keyHash[:8]),
			)

			ctx := gatewaycontext.SetConsumerInfo(r.Context(), gatewaycontext.ConsumerInfo{
				ConsumerID:      key.UserID.String(),
				APIKeyID:        key.ID.String(),
				IsAuthenticated: true,
			})

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
