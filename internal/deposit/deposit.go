package deposit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"

	"castellan/internal/repository/db"
	"castellan/internal/stellar"
)

const errKey = "error"

type Service struct {
	queries repository.Querier
	cfg     stellar.Config
}

func NewService(queries repository.Querier, cfg stellar.Config) *Service {
	return &Service{queries: queries, cfg: cfg}
}

func (s *Service) EnsureDepositMemo(ctx context.Context, userID uuid.UUID) (string, error) {
	entropy := ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0) //nolint:gosec // ULID entropy, not cryptographic
	newMemo := ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()

	memo, err := s.queries.EnsureUserDepositMemo(ctx, repository.EnsureUserDepositMemoParams{
		ID:      userID,
		NewMemo: newMemo,
	})
	if err != nil {
		return "", fmt.Errorf("ensure deposit memo: %w", err)
	}
	if !memo.Valid {
		return "", errors.New("deposit memo is null after upsert")
	}
	return memo.String, nil
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) //nolint:errchkjson
}
