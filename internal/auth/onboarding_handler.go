package auth

import (
	"log/slog"
	"net/http"
	"os"

	gatewaycontext "castellan/internal/gateway/context"

	"github.com/google/uuid"
)

func (h *Handler) CompleteOnboarding(w http.ResponseWriter, r *http.Request) {
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

	if _, err := h.queries.CompleteOnboarding(r.Context(), userID); err != nil {
		slog.ErrorContext(r.Context(), "failed to complete onboarding", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	apiBase := os.Getenv("API_BASE_URL")
	if apiBase == "" {
		apiBase = "http://localhost:8080"
	}
	writeJSON(w, http.StatusOK, map[string]string{"api_base_url": apiBase})
}
