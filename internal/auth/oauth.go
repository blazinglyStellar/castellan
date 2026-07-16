package auth

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"castellan/internal/provider"
	"castellan/internal/repository/db"

	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
)

const cookieStoreMaxAge = 86400 * 7 // 7 days in seconds

type OAuthHandler struct {
	pool           *pgxpool.Pool
	queries        repository.Querier
	sessionService *SessionService
	dashboardURL   string
}

func NewOAuthHandler(pool *pgxpool.Pool, queries repository.Querier, sessionService *SessionService, dashboardURL string) *OAuthHandler {
	return &OAuthHandler{
		pool:           pool,
		queries:        queries,
		sessionService: sessionService,
		dashboardURL:   dashboardURL,
	}
}

func InitGoth(_ string, apiBaseURL string) {
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	githubSecret := os.Getenv("GITHUB_CLIENT_SECRET")

	goth.UseProviders(
		google.New(googleClientID, googleSecret, apiBaseURL+"/auth/google/callback"),
		github.New(githubClientID, githubSecret, apiBaseURL+"/auth/github/callback"),
	)

	store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_STORE_SECRET")))
	store.MaxAge(cookieStoreMaxAge)
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	store.Options.Secure = false                  // TLS terminated at proxy — internal conn is HTTP
	store.Options.SameSite = http.SameSiteLaxMode // OAuth callback is a top-level GET redirect — Lax allows it

	gothic.Store = store
}

func (h *OAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	oauthProvider := r.PathValue("provider")

	if oauthProvider != "google" && oauthProvider != "github" {
		http.Error(w, "unsupported provider", http.StatusBadRequest)
		return
	}

	q := r.URL.Query()
	q.Set("provider", oauthProvider)
	r.URL.RawQuery = q.Encode()

	gothic.BeginAuthHandler(w, r)
}

func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	oauthProvider := r.PathValue("provider")
	if oauthProvider != "google" && oauthProvider != "github" {
		http.Error(w, "unsupported provider", http.StatusBadRequest)
		return
	}

	q := r.URL.Query()
	q.Set("provider", oauthProvider)
	r.URL.RawQuery = q.Encode()

	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		slog.ErrorContext(
			r.Context(), "oauth callback failed",
			slog.String("provider", oauthProvider),
			slog.String("error", err.Error()),
		)
		http.Error(w, "authentication failed", http.StatusUnauthorized)
		return
	}

	if user.Email == "" {
		slog.ErrorContext(
			r.Context(), "oauth provider returned no email",
			slog.String("provider", oauthProvider),
		)
		http.Error(w, "email required from provider", http.StatusUnauthorized)
		return
	}

	row, err := h.queries.UpsertUserByEmail(r.Context(), user.Email)
	if err != nil {
		slog.ErrorContext(
			r.Context(), "failed to upsert user",
			slog.String("error", err.Error()),
		)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if row.IsNew {
		if seedErr := provider.SeedHttpbinProvider(r.Context(), h.pool, h.queries); seedErr != nil {
			slog.ErrorContext(
				r.Context(), "failed to seed httpbin provider",
				slog.String("error", seedErr.Error()),
			)
		}
	}

	rawToken, err := h.sessionService.CreateSession(r.Context(), row.ID, 7*24*time.Hour)
	if err != nil {
		slog.ErrorContext(
			r.Context(), "failed to create session",
			slog.String("error", err.Error()),
		)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	SetSessionCookie(w, rawToken, 7*24*time.Hour)

	redirectPath := "/overview"
	if row.IsNew {
		redirectPath = "/onboarding"
	}
	http.Redirect(w, r, fmt.Sprintf("%s%s#session_token=%s", h.dashboardURL, redirectPath, rawToken), http.StatusFound)
}
