package auth

import (
	"log/slog"
	"net/http"
	"time"

	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/repository/db"

	"github.com/google/uuid"
)

type Handler struct {
	sessionService *SessionService
	queries        repository.Querier
}

func NewHandler(sessionService *SessionService, queries repository.Querier) *Handler {
	return &Handler{sessionService: sessionService, queries: queries}
}

type meResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: "authentication required"})
		return
	}

	userID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		slog.ErrorContext(r.Context(), "invalid consumer id", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "invalid identity"})
		return
	}

	user, err := h.queries.GetUserByID(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to lookup user", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, meResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	})
}

type DashboardMeResponse struct {
	ID                   string `json:"id"`
	Email                string `json:"email"`
	Role                 string `json:"role"`
	DepositMemo          string `json:"deposit_memo"`
	PayoutStellarAddress string `json:"payout_stellar_address,omitempty"`
}

func (h *Handler) DashboardMe(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: "authentication required"})
		return
	}

	userID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		slog.ErrorContext(r.Context(), "invalid consumer id", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "invalid identity"})
		return
	}

	user, err := h.queries.GetUserByID(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to lookup user", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "internal error"})
		return
	}

	count, err := h.queries.GetUserProviderCount(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to check provider count", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "internal error"})
		return
	}

	role := "consumer"
	if count > 0 {
		role = "provider"
	}

	depositMemo := ""
	if user.DepositMemo.Valid {
		depositMemo = user.DepositMemo.String
	}

	payoutAddr := ""
	if user.PayoutStellarAddress.Valid {
		payoutAddr = user.PayoutStellarAddress.String
	}

	writeJSON(w, http.StatusOK, DashboardMeResponse{
		ID:                   user.ID.String(),
		Email:                user.Email,
		Role:                 role,
		DepositMemo:          depositMemo,
		PayoutStellarAddress: payoutAddr,
	})
}

type logoutResponse struct {
	Message string `json:"message"`
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	rawToken, err := ReadSessionCookie(r)
	if err != nil {
		writeJSON(w, http.StatusOK, logoutResponse{Message: "logged out"})
		return
	}

	if err := h.sessionService.RevokeSession(r.Context(), rawToken); err != nil {
		slog.WarnContext(
			r.Context(), "session revoke failed",
			slog.String("error", err.Error()),
		)
	}

	ClearSessionCookie(w)

	writeJSON(w, http.StatusOK, logoutResponse{Message: "logged out"})
}
