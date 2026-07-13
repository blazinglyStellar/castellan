package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// keyBytes is the random-bytes length used to derive new API keys.
// API_KEY_BYTES env var (see .env.example) is the planned override;
// hard-coded for now (GH #55).
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

// ListKeys returns all API keys for the given user.
// Only metadata is returned — key hashes are never exposed.
func (s *KeyService) ListKeys(ctx context.Context, userID uuid.UUID) ([]ListKeysItem, error) {
	keys, err := s.queries.ListKeysByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list keys: %w", err)
	}

	items := make([]ListKeysItem, 0, len(keys))
	for _, k := range keys {
		items = append(items, ListKeysItem{
			ID:        k.ID,
			Label:     k.Label,
			Status:    k.Status,
			CreatedAt: k.CreatedAt,
			ExpiresAt: k.ExpiresAt,
		})
	}

	return items, nil
}

// GetKeyByID fetches a key by ID and verifies it belongs to the given user.
func (s *KeyService) GetKeyByID(ctx context.Context, keyID, userID uuid.UUID) (repository.ApiKey, error) {
	key, err := s.queries.GetKeyByID(ctx, keyID)
	if err != nil {
		return repository.ApiKey{}, fmt.Errorf("key not found: %w", err)
	}
	if key.UserID != userID {
		return repository.ApiKey{}, fmt.Errorf("key not found: %w", err)
	}
	return key, nil
}

// RevokeKey sets the key's status to revoked using an atomic SQL guard
// (only updates if status is not already revoked). Returns error if already revoked.
func (s *KeyService) RevokeKey(ctx context.Context, keyID, userID uuid.UUID) (repository.ApiKey, error) {
	if _, err := s.GetKeyByID(ctx, keyID, userID); err != nil {
		return repository.ApiKey{}, err
	}

	updated, err := s.queries.RevokeKey(ctx, keyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repository.ApiKey{}, errors.New("key already revoked")
		}
		return repository.ApiKey{}, fmt.Errorf("revoke key: %w", err)
	}
	return updated, nil
}

// RotateKey creates a replacement key with the same label and expiration, then
// revokes the old key. Generation happens first so the old key is untouched if
// creation fails. Returns error if the key is not active.
func (s *KeyService) RotateKey(ctx context.Context, keyID, userID uuid.UUID) (rawKey string, newKey repository.ApiKey, err error) {
	key, err := s.GetKeyByID(ctx, keyID, userID)
	if err != nil {
		return "", repository.ApiKey{}, err
	}
	if key.Status != repository.ApiKeyStatusActive {
		return "", repository.ApiKey{}, errors.New("key is not active")
	}

	var expiresAt *time.Time
	if key.ExpiresAt.Valid {
		expiresAt = &key.ExpiresAt.Time
	}

	rawKey, newKey, err = s.GenerateKey(ctx, userID, key.Label.String, expiresAt)
	if err != nil {
		return "", repository.ApiKey{}, fmt.Errorf("generate replacement key: %w", err)
	}

	if _, err := s.queries.RevokeKey(ctx, keyID); err != nil {
		return "", repository.ApiKey{}, fmt.Errorf("revoke old key: %w", err)
	}

	return rawKey, newKey, nil
}

// UpdateKeyLabel updates the label of a key after verifying ownership.
func (s *KeyService) UpdateKeyLabel(ctx context.Context, keyID, userID uuid.UUID, label string) (repository.ApiKey, error) {
	if _, err := s.GetKeyByID(ctx, keyID, userID); err != nil {
		return repository.ApiKey{}, err
	}

	updated, err := s.queries.UpdateKeyLabel(ctx, repository.UpdateKeyLabelParams{
		ID:    keyID,
		Label: pgtype.Text{String: label, Valid: label != ""},
	})
	if err != nil {
		return repository.ApiKey{}, fmt.Errorf("update key label: %w", err)
	}
	return updated, nil
}

// UpdateKeyExpiration updates the expiration of a key after verifying ownership.
func (s *KeyService) UpdateKeyExpiration(ctx context.Context, keyID, userID uuid.UUID, expiresAt *time.Time) (repository.ApiKey, error) {
	if _, err := s.GetKeyByID(ctx, keyID, userID); err != nil {
		return repository.ApiKey{}, err
	}

	exp := pgtype.Timestamptz{Valid: expiresAt != nil}
	if expiresAt != nil {
		exp.Time = *expiresAt
	}

	updated, err := s.queries.UpdateKeyExpiration(ctx, repository.UpdateKeyExpirationParams{
		ID:        keyID,
		ExpiresAt: exp,
	})
	if err != nil {
		return repository.ApiKey{}, fmt.Errorf("update key expiration: %w", err)
	}
	return updated, nil
}

// HashKey returns the SHA-256 hex digest of the given key.
func HashKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(hash[:])
}
