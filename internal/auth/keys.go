package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const keyBytes = 32

type KeyService struct {
	queries repository.Querier
}

func NewKeyService(queries repository.Querier) *KeyService {
	return &KeyService{queries: queries}
}

func (s *KeyService) GenerateKey(ctx context.Context, userID uuid.UUID, label string, expiresAt *time.Time) (rawKey string, err error) {
	key := make([]byte, keyBytes)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}

	rawKey = "ca_" + base64.RawURLEncoding.EncodeToString(key)
	keyHash := HashKey(rawKey)

	params := repository.InsertKeyParams{
		UserID:    userID,
		KeyHash:   keyHash,
		Label:     pgtype.Text{String: label, Valid: label != ""},
		Status:    repository.ApiKeyStatusActive,
		CreatedAt: time.Now().UTC(),
	}

	if expiresAt != nil {
		params.ExpiresAt = pgtype.Timestamptz{Time: *expiresAt, Valid: true}
	}

	if _, err := s.queries.InsertKey(ctx, params); err != nil {
		return "", fmt.Errorf("persist key hash: %w", err)
	}

	return rawKey, nil
}

func HashKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(hash[:])
}
