package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
