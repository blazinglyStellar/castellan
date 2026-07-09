package usage

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	gatewaycontext "castellan/internal/gateway/context"

	"github.com/google/uuid"
)

const errKey = "error"

const (
	defaultLimit     = 50
	maxLimit         = 100
	cursorPartsCount = 2
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListUsage(w http.ResponseWriter, r *http.Request) {
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

	role := r.URL.Query().Get("role")
	limit := parseLimit(r.URL.Query().Get("limit"))

	filter := parseUsageFilter(r)
	cursorTs, cursorID := parseCursor(r.URL.Query().Get("cursor"))

	var resp *UsageListResponse
	if role == "provider" {
		resp, err = h.service.ListUsageByProvider(r.Context(), userID, filter, cursorTs, cursorID, limit)
	} else {
		resp, err = h.service.ListUsageByConsumer(r.Context(), userID, filter, cursorTs, cursorID, limit)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to list usage events"})
		return
	}

	if resp == nil {
		resp = &UsageListResponse{Data: []UsageEventItem{}}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetEarnings(w http.ResponseWriter, r *http.Request) {
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

	startDate := parseTimeParam(r.URL.Query().Get("start_date"))
	endDate := parseTimeParam(r.URL.Query().Get("end_date"))

	resp, err := h.service.GetEarnings(r.Context(), userID, startDate, endDate)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to get earnings"})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func parseLimit(s string) int32 {
	if s == "" {
		return defaultLimit
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil || v < 1 {
		return defaultLimit
	}
	if v > maxLimit {
		return maxLimit
	}
	return int32(v)
}

func parseUsageFilter(r *http.Request) UsageFilter {
	q := r.URL.Query()

	var providerID uuid.UUID
	if p := q.Get("provider_id"); p != "" {
		if id, err := uuid.Parse(p); err == nil {
			providerID = id
		}
	}

	var endpointID uuid.UUID
	if e := q.Get("endpoint_id"); e != "" {
		if id, err := uuid.Parse(e); err == nil {
			endpointID = id
		}
	}
	var statusCode int32
	if sc := q.Get("status_code"); sc != "" {
		if v, err := strconv.ParseInt(sc, 10, 32); err == nil {
			statusCode = int32(v)
		}
	}

	startDate := time.Time{}
	if sd := q.Get("start_date"); sd != "" {
		if t, err := time.Parse(time.RFC3339, sd); err == nil {
			startDate = t
		}
	}

	endDate := time.Time{}
	if ed := q.Get("end_date"); ed != "" {
		if t, err := time.Parse(time.RFC3339, ed); err == nil {
			endDate = t
		}
	}

	hasFilters := q.Get("provider_id") != "" || q.Get("endpoint_id") != "" ||
		q.Get("status_code") != "" || q.Get("start_date") != "" || q.Get("end_date") != ""

	return UsageFilter{
		ProviderID: providerID,
		EndpointID: endpointID,
		StatusCode: statusCode,
		StartDate:  startDate,
		EndDate:    endDate,
		HasFilters: hasFilters,
	}
}

func parseCursor(s string) (time.Time, uuid.UUID) {
	if s == "" {
		return time.Time{}, uuid.UUID{}
	}
	parts := strings.SplitN(s, "|", cursorPartsCount)
	if len(parts) != cursorPartsCount {
		return time.Time{}, uuid.UUID{}
	}
	t, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return time.Time{}, uuid.UUID{}
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.UUID{}
	}
	return t, id
}

func parseTimeParam(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) //nolint:errchkjson
}
