package deposit

import (
	"net/http"

	"github.com/google/uuid"

	gatewaycontext "castellan/internal/gateway/context"
)

type IntentResponse struct {
	SEP7URI     string `json:"sep7_uri"`
	QRCode      string `json:"qr_code"`
	Memo        string `json:"memo"`
	Destination string `json:"destination"`
	MinAmount   string `json:"minimum_amount"`
	Asset       string `json:"asset"`
}

func (h *Handler) DepositIntent(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: "authentication required"})
		return
	}

	userID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "invalid consumer identity"})
		return
	}

	memo, err := h.service.EnsureDepositMemo(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to generate deposit memo"})
		return
	}

	minAmount := h.service.cfg.MinDepositAmount.String()
	destination := h.service.cfg.HotWalletAddress
	asset := "XLM"

	uri := BuildSEP7URI(destination, memo, minAmount, asset)
	qrCode, err := GenerateQRCode(uri)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to generate QR code"})
		return
	}

	writeJSON(w, http.StatusOK, IntentResponse{
		SEP7URI:     uri,
		QRCode:      qrCode,
		Memo:        memo,
		Destination: destination,
		MinAmount:   minAmount,
		Asset:       asset,
	})
}
