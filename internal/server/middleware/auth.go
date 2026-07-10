package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"castellan/internal/auth"
	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/repository/db"

	"github.com/jackc/pgx/v5"
)

type SessionValidator interface {
	ValidateSession(ctx context.Context, rawToken string) (*repository.SessionToken, error)
}

type SessionValidatorFunc func(ctx context.Context, rawToken string) (*repository.SessionToken, error)

func (f SessionValidatorFunc) ValidateSession(ctx context.Context, rawToken string) (*repository.SessionToken, error) {
	return f(ctx, rawToken)
}

const bearerPrefix = "Bearer "

type KeyValidator interface {
	ValidateKey(ctx context.Context, keyHash string) (repository.ApiKey, error)
}

type KeyValidatorFunc func(ctx context.Context, keyHash string) (repository.ApiKey, error)

func (f KeyValidatorFunc) ValidateKey(ctx context.Context, keyHash string) (repository.ApiKey, error) {
	return f(ctx, keyHash)
}

func AuthCheck(validator KeyValidator, sessionValidator SessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if consumer := gatewaycontext.GetConsumerInfo(r.Context()); consumer.IsAuthenticated {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")

			if authHeader != "" && strings.HasPrefix(authHeader, bearerPrefix) {
				rawKey := strings.TrimPrefix(authHeader, bearerPrefix)

				if strings.HasPrefix(rawKey, "ca_") {
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
					return
				}

				token, err := sessionValidator.ValidateSession(r.Context(), rawKey)
				if err == nil {
					ctx := gatewaycontext.SetConsumerInfo(r.Context(), gatewaycontext.ConsumerInfo{
						ConsumerID:      token.UserID.String(),
						IsAuthenticated: true,
					})
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				switch {
				case errors.Is(err, auth.ErrSessionNotFound):
					writeJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{errKey: "invalid api key"})
				case errors.Is(err, auth.ErrSessionNotActive):
					writeJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{errKey: "session token revoked"})
				case errors.Is(err, auth.ErrSessionExpired):
					writeJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{errKey: "session token expired"})
				default:
					slog.ErrorContext(r.Context(), "session validation failed", slog.String("error", err.Error()))
					writeJSON(r.Context(), w, http.StatusInternalServerError, map[string]string{errKey: "internal error"})
				}
				return
			}

			rawToken, err := auth.ReadSessionCookie(r)
			if err != nil {
				writeJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{errKey: "authentication required"})
				return
			}

			token, err := sessionValidator.ValidateSession(r.Context(), rawToken)
			if err != nil {
				switch {
				case errors.Is(err, auth.ErrSessionNotFound):
					writeJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{errKey: "invalid session"})
				case errors.Is(err, auth.ErrSessionNotActive):
					writeJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{errKey: "session revoked"})
				case errors.Is(err, auth.ErrSessionExpired):
					writeJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{errKey: "session expired"})
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
				IsAuthenticated: true,
			})

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
