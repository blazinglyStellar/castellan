package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

const internalServerErrorMessage = "internal server error"

// Recovery catches panics from downstream handlers, logs the stack trace, and returns 500.
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func(ctx context.Context) {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(
						ctx,
						"handler panic recovered",
						slog.Any("panic", recovered),
						slog.String("stack", string(debug.Stack())),
					)
					if err := writeInternalServerError(w); err != nil {
						logger.ErrorContext(
							ctx,
							"write recovery response failed",
							slog.Any("error", err),
						)
					}
				}
			}(r.Context())

			next.ServeHTTP(w, r)
		})
	}
}

func writeInternalServerError(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)

	if err := json.NewEncoder(w).Encode(map[string]string{
		errKey: internalServerErrorMessage,
	}); err != nil {
		return fmt.Errorf("encode recovery response: %w", err)
	}

	return nil
}
