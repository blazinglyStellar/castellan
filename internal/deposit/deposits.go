package deposit

import "net/http"

func (h *Handler) ListDeposits(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"deposits": []any{}})
}
