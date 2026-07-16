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
	isProd := os.Getenv("APP_ENV") == "production"
	secure := isProd // production: TLS terminated at proxy, Secure=true for SameSite=None

	http.SetCookie(w, &http.Cookie{ // #nosec G124
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().UTC().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteNoneMode,
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	isProd := os.Getenv("APP_ENV") == "production"

	http.SetCookie(w, &http.Cookie{ // #nosec G124
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteNoneMode,
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
