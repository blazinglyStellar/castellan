package auth

import (
	"errors"
	"net/http"
	"time"
)

const sessionCookieName = "session_token"

var errCookieNotFound = errors.New("session cookie not found")

func SetSessionCookie(w http.ResponseWriter, token string, ttl time.Duration) {
	secure := false // TLS terminated at proxy — internal conn is HTTP

	http.SetCookie(w, &http.Cookie{ // #nosec G124
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
	http.SetCookie(w, &http.Cookie{ // #nosec G124
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false, // TLS terminated at proxy
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
