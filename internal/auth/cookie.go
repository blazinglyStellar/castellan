package auth

import (
	"errors"
	"net/http"
	"os"
	"time"
)

const sessionCookieName = "session_token"

var errCookieNotFound = errors.New("session cookie not found")

func SetSessionCookie(w http.ResponseWriter, token string, ttl time.Duration) {
	secure := os.Getenv("APP_ENV") != "local"

	http.SetCookie(w, &http.Cookie{ //nolint:gosec // Secure set dynamically based on APP_ENV
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().UTC().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // Secure set dynamically based on APP_ENV
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   os.Getenv("APP_ENV") != "local",
		SameSite: http.SameSiteLaxMode,
	})
}

func ReadSessionCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", errCookieNotFound
	}

	if cookie.Value == "" {
		return "", errCookieNotFound
	}

	return cookie.Value, nil
}
