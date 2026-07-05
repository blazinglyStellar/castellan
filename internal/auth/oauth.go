package auth

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"castellan/internal/repository/db"

	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
)

type OAuthHandler struct {
	queries        repository.Querier
	sessionService *SessionService
	dashboardURL   string
}

func NewOAuthHandler(queries repository.Querier, sessionService *SessionService, dashboardURL string) *OAuthHandler {
	return &OAuthHandler{
		queries:        queries,
		sessionService: sessionService,
		dashboardURL:   dashboardURL,
	}
}

func InitGoth(dashboardURL string) {
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	githubSecret := os.Getenv("GITHUB_CLIENT_SECRET")

	goth.UseProviders(
		google.New(googleClientID, googleSecret, dashboardURL+"/auth/google/callback"),
		github.New(githubClientID, githubSecret, dashboardURL+"/auth/github/callback"),
	)

	store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_STORE_SECRET")))
	store.MaxAge(86400 * 7)
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	store.Options.Secure = os.Getenv("APP_ENV") != "local"

	gothic.Store = store
}

func (h *OAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")

	if provider != "google" && provider != "github" {
		http.Error(w, "unsupported provider", http.StatusBadRequest)
		return
	}

	gothic.BeginAuthHandler(w, r)
}

func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	if provider != "google" && provider != "github" {
		http.Error(w, "unsupported provider", http.StatusBadRequest)
		return
	}

	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		slog.ErrorContext(
			r.Context(), "oauth callback failed",
			slog.String("provider", provider),
			slog.String("error", err.Error()),
		)
		http.Error(w, "authentication failed", http.StatusUnauthorized)
		return
	}

	if user.Email == "" {
		slog.ErrorContext(
			r.Context(), "oauth provider returned no email",
			slog.String("provider", provider),
		)
		http.Error(w, "email required from provider", http.StatusUnauthorized)
		return
	}

	dbUser, err := h.queries.UpsertUserByEmail(r.Context(), user.Email)
	if err != nil {
		slog.ErrorContext(
			r.Context(), "failed to upsert user",
			slog.String("error", err.Error()),
		)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	rawToken, err := h.sessionService.CreateSession(r.Context(), dbUser.ID, 7*24*time.Hour)
	if err != nil {
		slog.ErrorContext(
			r.Context(), "failed to create session",
			slog.String("error", err.Error()),
		)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	SetSessionCookie(w, rawToken, 7*24*time.Hour)

	http.Redirect(w, r, h.dashboardURL+"/dashboard", http.StatusFound)
}
