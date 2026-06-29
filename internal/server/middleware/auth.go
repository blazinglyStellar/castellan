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

// SessionValidator validates a session token (st_ prefix).
type SessionValidator interface {
	ValidateSession(ctx context.Context, rawToken string) (*repository.SessionToken, error)
}

// SessionValidatorFunc is an adapter that lets a function serve as a SessionValidator.
type SessionValidatorFunc func(ctx context.Context, rawToken string) (*repository.SessionToken, error)

// ValidateSession delegates to the underlying function.
func (f SessionValidatorFunc) ValidateSession(ctx context.Context, rawToken string) (*repository.SessionToken, error) {
	return f(ctx, rawToken)
}

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

// AuthCheck validates the Authorization: Bearer header on every request.
// It accepts both ca_ (API key) and st_ (session token) prefixed tokens,
// validates them, and injects ConsumerInfo into the context on success.
// The raw credential is never logged.
func AuthCheck(validator KeyValidator, sessionValidator SessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			if !strings.HasPrefix(authHeader, bearerPrefix) {
				writeJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{errKey: "missing authorization header"})
				return
			}

			rawKey := strings.TrimPrefix(authHeader, bearerPrefix)

			switch {
			case strings.HasPrefix(rawKey, "ca_"):
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

				if key.ExpiresAt.Valid && !key.ExpiresAt.Time.After(time.Now()) {
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

			case strings.HasPrefix(rawKey, "st_"):
				token, err := sessionValidator.ValidateSession(r.Context(), rawKey)
				if err != nil {
					switch {
					case errors.Is(err, auth.ErrSessionTokenNotFound):
						writeJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{errKey: "invalid session token"})
					case errors.Is(err, auth.ErrSessionTokenNotActive):
						writeJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{errKey: "session token revoked"})
					case errors.Is(err, auth.ErrSessionTokenExpired):
						writeJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{errKey: "session token expired"})
					default:
						slog.ErrorContext(
							r.Context(), "session validation failed",
							slog.String("error", err.Error()),
						)
						writeJSON(r.Context(), w, http.StatusInternalServerError, map[string]string{errKey: "internal error"})
					}
					return
				}

				slog.DebugContext(
					r.Context(), "authenticated request",
					slog.String("consumer_id", token.UserID.String()),
					slog.String("session_token_id", token.ID.String()),
				)

				ctx := gatewaycontext.SetConsumerInfo(r.Context(), gatewaycontext.ConsumerInfo{
					ConsumerID:      token.UserID.String(),
					APIKeyID:        token.ID.String(),
					IsAuthenticated: true,
				})

				next.ServeHTTP(w, r.WithContext(ctx))

			default:
				writeJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{errKey: "invalid api key"})
			}
		})
	}
}
