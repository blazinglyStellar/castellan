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

// KeyService generates, hashes, and persists API keys.
type KeyService struct {
	queries repository.Querier
}

// NewKeyService returns a KeyService backed by the given repository.
func NewKeyService(queries repository.Querier) *KeyService {
	return &KeyService{queries: queries}
}

// GenerateKey creates a new API key (ca_ prefix, 32 random bytes, base64),
// persists only its SHA-256 hash, and returns the raw key along with the
// persisted ApiKey record (which includes the id, status, and timestamps).
func (s *KeyService) GenerateKey(ctx context.Context, userID uuid.UUID, label string, expiresAt *time.Time) (rawKey string, key repository.ApiKey, err error) {
	raw := make([]byte, keyBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", repository.ApiKey{}, fmt.Errorf("generate random bytes: %w", err)
	}

	rawKey = "ca_" + base64.RawURLEncoding.EncodeToString(raw)
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

	key, err = s.queries.InsertKey(ctx, params)
	if err != nil {
		return "", repository.ApiKey{}, fmt.Errorf("persist key hash: %w", err)
	}

	return rawKey, key, nil
}

// HashKey returns the SHA-256 hex digest of the given key.
func HashKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(hash[:])
}
