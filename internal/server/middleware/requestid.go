package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type requestIDKey struct{}

var requestIDContextKey = requestIDKey{}

// RequestID is middleware that injects a request ID into the context and
// response header. It reads X-Request-ID from the incoming request; if empty,
// generates a UUID v4.
func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.NewString()
			}

			ctx := context.WithValue(r.Context(), requestIDContextKey, requestID)
			r = r.WithContext(ctx)

			w.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(w, r)
		})
	}
}

// GetRequestID returns the request ID from ctx, or empty string if not set.
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, ok := ctx.Value(requestIDContextKey).(string)
	if !ok {
		return ""
	}
	return requestID
}