package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type mockQuerier struct {
	repository.Querier
	insertedHash string
	insertCalled bool

	mu         sync.Mutex
	keysByUser map[uuid.UUID][]repository.ApiKey
}

func (m *mockQuerier) GetKeyByID(ctx context.Context, id uuid.UUID) (repository.ApiKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, keys := range m.keysByUser {
		for _, k := range keys {
			if k.ID == id {
				return k, nil
			}
		}
	}
	return repository.ApiKey{}, errors.New("key not found")
}

func (m *mockQuerier) UpdateKeyStatus(ctx context.Context, arg repository.UpdateKeyStatusParams) (repository.ApiKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for userID, keys := range m.keysByUser {
		for i, k := range keys {
			if k.ID == arg.ID {
				k.Status = arg.Status
				m.keysByUser[userID][i] = k
				return k, nil
			}
		}
	}
	return repository.ApiKey{}, errors.New("key not found")
}

func (m *mockQuerier) InsertKey(ctx context.Context, arg repository.InsertKeyParams) (repository.ApiKey, error) {
	m.insertCalled = true
	m.insertedHash = arg.KeyHash
	return repository.ApiKey{
		ID:        uuid.New(),
		UserID:    arg.UserID,
		KeyHash:   arg.KeyHash,
		Label:     arg.Label,
		Status:    repository.ApiKeyStatusActive,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: arg.ExpiresAt,
	}, nil
}

func TestGenerateKey_Format(t *testing.T) {
	s := NewKeyService(&mockQuerier{})
	rawKey, _, err := s.GenerateKey(context.Background(), uuid.New(), "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rawKey, "ca_") {
		t.Errorf("key should start with ca_, got %q", rawKey)
	}
	if len(rawKey) < 40 {
		t.Errorf("key length should be >= 40, got %d", len(rawKey))
	}
}

func TestGenerateKey_Uniqueness(t *testing.T) {
	s := NewKeyService(&mockQuerier{})
	ctx := context.Background()
	uid := uuid.New()
	k1, _, err := s.GenerateKey(ctx, uid, "key-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	k2, _, err := s.GenerateKey(ctx, uid, "key-2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if k1 == k2 {
		t.Error("two consecutive GenerateKey calls produced the same key")
	}
}

func TestHashKey_Deterministic(t *testing.T) {
	input := "ca_test-key-value"
	h1 := HashKey(input)
	h2 := HashKey(input)
	if h1 != h2 {
		t.Error("HashKey is not deterministic")
	}
	hash := sha256.Sum256([]byte(input))
	expected := hex.EncodeToString(hash[:])
	if h1 != expected {
		t.Errorf("HashKey output does not match expected hash:\n  got:  %s\n  want: %s", h1, expected)
	}
}

func (m *mockQuerier) ListKeysByUser(ctx context.Context, userID uuid.UUID) ([]repository.ApiKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.keysByUser == nil {
		return []repository.ApiKey{}, nil
	}
	return m.keysByUser[userID], nil
}

func TestListKeys_ReturnsKeys(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()
	keys := []repository.ApiKey{
		{
			ID:        uuid.New(),
			UserID:    userID,
			KeyHash:   "should-not-be-exposed",
			Label:     pgtype.Text{String: "Production", Valid: true},
			Status:    repository.ApiKeyStatusActive,
			CreatedAt: now,
			ExpiresAt: pgtype.Timestamptz{Time: now.Add(24 * time.Hour), Valid: true},
		},
		{
			ID:        uuid.New(),
			UserID:    userID,
			KeyHash:   "also-not-exposed",
			Label:     pgtype.Text{String: "Staging", Valid: true},
			Status:    repository.ApiKeyStatusRevoked,
			CreatedAt: now.Add(-time.Hour),
		},
	}

	mq := &mockQuerier{keysByUser: map[uuid.UUID][]repository.ApiKey{userID: keys}}
	s := NewKeyService(mq)
	items, err := s.ListKeys(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(items))
	}
	if items[0].ID != keys[0].ID {
		t.Error("first key ID mismatch")
	}
	if items[1].Status != repository.ApiKeyStatusRevoked {
		t.Errorf("expected revoked status, got %s", items[1].Status)
	}
}

func TestListKeysService_Empty(t *testing.T) {
	mq := &mockQuerier{}
	s := NewKeyService(mq)
	items, err := s.ListKeys(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if items == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(items) != 0 {
		t.Errorf("expected 0 keys, got %d", len(items))
	}
}

func TestRevokeKey_Success(t *testing.T) {
	userID := uuid.New()
	keyID := uuid.New()
	now := time.Now().UTC()
	mq := &mockQuerier{
		keysByUser: map[uuid.UUID][]repository.ApiKey{
			userID: {
				{
					ID:        keyID,
					UserID:    userID,
					KeyHash:   "hash",
					Label:     pgtype.Text{String: "test", Valid: true},
					Status:    repository.ApiKeyStatusActive,
					CreatedAt: now,
				},
			},
		},
	}
	s := NewKeyService(mq)
	key, err := s.RevokeKey(context.Background(), keyID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if key.Status != repository.ApiKeyStatusRevoked {
		t.Errorf("expected revoked, got %s", key.Status)
	}
}

func TestRevokeKey_AlreadyRevoked(t *testing.T) {
	userID := uuid.New()
	keyID := uuid.New()
	mq := &mockQuerier{
		keysByUser: map[uuid.UUID][]repository.ApiKey{
			userID: {
				{
					ID:     keyID,
					UserID: userID,
					Status: repository.ApiKeyStatusRevoked,
				},
			},
		},
	}
	s := NewKeyService(mq)
	_, err := s.RevokeKey(context.Background(), keyID, userID)
	if err == nil {
		t.Fatal("expected error for already revoked key")
	}
}

func TestRevokeKey_NotFound(t *testing.T) {
	s := NewKeyService(&mockQuerier{})
	_, err := s.RevokeKey(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent key")
	}
}

func TestRevokeKey_NotOwner(t *testing.T) {
	ownerID := uuid.New()
	otherUser := uuid.New()
	keyID := uuid.New()
	mq := &mockQuerier{
		keysByUser: map[uuid.UUID][]repository.ApiKey{
			ownerID: {
				{
					ID:     keyID,
					UserID: ownerID,
					Status: repository.ApiKeyStatusActive,
				},
			},
		},
	}
	s := NewKeyService(mq)
	_, err := s.RevokeKey(context.Background(), keyID, otherUser)
	if err == nil {
		t.Fatal("expected error for non-owner")
	}
}

func TestRotateKey_Success(t *testing.T) {
	userID := uuid.New()
	keyID := uuid.New()
	now := time.Now().UTC()
	future := now.Add(30 * 24 * time.Hour)
	mq := &mockQuerier{
		keysByUser: map[uuid.UUID][]repository.ApiKey{
			userID: {
				{
					ID:        keyID,
					UserID:    userID,
					KeyHash:   "old-hash",
					Label:     pgtype.Text{String: "Production key", Valid: true},
					Status:    repository.ApiKeyStatusActive,
					CreatedAt: now,
					ExpiresAt: pgtype.Timestamptz{Time: future, Valid: true},
				},
			},
		},
	}
	s := NewKeyService(mq)
	rawKey, newKey, err := s.RotateKey(context.Background(), keyID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rawKey, "ca_") {
		t.Errorf("expected raw key to start with ca_, got %q", rawKey)
	}
	if newKey.Status != repository.ApiKeyStatusActive {
		t.Errorf("expected new key status active, got %s", newKey.Status)
	}
	if newKey.Label.String != "Production key" {
		t.Errorf("expected label 'Production key', got %q", newKey.Label.String)
	}
}

func TestRotateKey_HashesDiffer(t *testing.T) {
	userID := uuid.New()
	keyID := uuid.New()
	mq := &mockQuerier{
		keysByUser: map[uuid.UUID][]repository.ApiKey{
			userID: {
				{
					ID:        keyID,
					UserID:    userID,
					KeyHash:   "old-hash",
					Label:     pgtype.Text{String: "test", Valid: true},
					Status:    repository.ApiKeyStatusActive,
					CreatedAt: time.Now().UTC(),
				},
			},
		},
	}
	s := NewKeyService(mq)
	_, newKey, err := s.RotateKey(context.Background(), keyID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if newKey.KeyHash == "old-hash" {
		t.Error("old and new key hashes should differ")
	}
}

func TestRotateKey_NotFound(t *testing.T) {
	s := NewKeyService(&mockQuerier{})
	_, _, err := s.RotateKey(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent key")
	}
}

func TestRotateKey_NotOwner(t *testing.T) {
	ownerID := uuid.New()
	otherUser := uuid.New()
	keyID := uuid.New()
	mq := &mockQuerier{
		keysByUser: map[uuid.UUID][]repository.ApiKey{
			ownerID: {
				{
					ID:     keyID,
					UserID: ownerID,
					Status: repository.ApiKeyStatusActive,
				},
			},
		},
	}
	s := NewKeyService(mq)
	_, _, err := s.RotateKey(context.Background(), keyID, otherUser)
	if err == nil {
		t.Fatal("expected error for non-owner")
	}
}

func TestGenerateKey_PersistsHash(t *testing.T) {
	mq := &mockQuerier{}
	s := NewKeyService(mq)
	key, _, err := s.GenerateKey(context.Background(), uuid.New(), "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !mq.insertCalled {
		t.Fatal("InsertKey was not called")
	}
	expectedHash := HashKey(key)
	if mq.insertedHash != expectedHash {
		t.Errorf("InsertKey called with hash %q, expected %q", mq.insertedHash, expectedHash)
	}
}
