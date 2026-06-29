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
	"github.com/jackc/pgx/v5"
)

type mockSessionQuerier struct {
	repository.Querier

	mu             sync.Mutex
	sessionsByUser map[uuid.UUID][]repository.SessionToken
	insertedHash   string
	insertCalled   bool
}

func (m *mockSessionQuerier) InsertSessionToken(
	ctx context.Context,
	arg repository.InsertSessionTokenParams,
) (repository.SessionToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.insertCalled = true
	m.insertedHash = arg.TokenHash

	token := repository.SessionToken{
		ID:        uuid.New(),
		UserID:    arg.UserID,
		TokenHash: arg.TokenHash,
		Label:     arg.Label,
		Scope:     arg.Scope,
		Status:    repository.SessionTokenStatusActive,
		ExpiresAt: arg.ExpiresAt,
		CreatedAt: time.Now().UTC(),
	}

	if m.sessionsByUser == nil {
		m.sessionsByUser = make(map[uuid.UUID][]repository.SessionToken)
	}
	m.sessionsByUser[arg.UserID] = append(m.sessionsByUser[arg.UserID], token)

	return token, nil
}

func (m *mockSessionQuerier) GetSessionTokenByHash(
	ctx context.Context,
	tokenHash string,
) (repository.SessionToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, tokens := range m.sessionsByUser {
		for _, t := range tokens {
			if t.TokenHash == tokenHash {
				return t, nil
			}
		}
	}
	return repository.SessionToken{}, pgx.ErrNoRows
}

func (m *mockSessionQuerier) RevokeSessionToken(
	ctx context.Context,
	id uuid.UUID,
) (repository.SessionToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for userID, tokens := range m.sessionsByUser {
		for i, t := range tokens {
			if t.ID == id {
				if t.Status != repository.SessionTokenStatusActive {
					return repository.SessionToken{}, pgx.ErrNoRows
				}
				t.Status = repository.SessionTokenStatusRevoked
				m.sessionsByUser[userID][i] = t
				return t, nil
			}
		}
	}
	return repository.SessionToken{}, pgx.ErrNoRows
}

func (m *mockSessionQuerier) UpdateSessionTokenStatus(
	ctx context.Context,
	arg repository.UpdateSessionTokenStatusParams,
) (repository.SessionToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for userID, tokens := range m.sessionsByUser {
		for i, t := range tokens {
			if t.ID == arg.ID {
				t.Status = arg.Status
				m.sessionsByUser[userID][i] = t
				return t, nil
			}
		}
	}
	return repository.SessionToken{}, pgx.ErrNoRows
}

func TestGenerateSessionToken_Format(t *testing.T) {
	s := NewSessionService(&mockSessionQuerier{})
	rawToken, err := s.GenerateSessionToken(
		context.Background(),
		uuid.New(),
		"test",
		nil,
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rawToken, "st_") {
		t.Errorf("token should start with st_, got %q", rawToken)
	}
	if len(rawToken) < 40 {
		t.Errorf("token length should be >= 40, got %d", len(rawToken))
	}
}

func TestGenerateSessionToken_StoresHashOnly(t *testing.T) {
	mq := &mockSessionQuerier{}
	s := NewSessionService(mq)
	rawToken, err := s.GenerateSessionToken(
		context.Background(),
		uuid.New(),
		"test",
		nil,
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !mq.insertCalled {
		t.Fatal("InsertSessionToken was not called")
	}
	expectedHash := HashToken(rawToken)
	if mq.insertedHash != expectedHash {
		t.Errorf("InsertSessionToken called with hash %q, expected %q", mq.insertedHash, expectedHash)
	}
}

func TestGenerateSessionToken_Uniqueness(t *testing.T) {
	s := NewSessionService(&mockSessionQuerier{})
	ctx := context.Background()
	uid := uuid.New()

	t1, err := s.GenerateSessionToken(ctx, uid, "test-1", nil, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	t2, err := s.GenerateSessionToken(ctx, uid, "test-2", nil, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if t1 == t2 {
		t.Error("two consecutive GenerateSessionToken calls produced the same token")
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	input := "st_test-token-value"
	h1 := HashToken(input)
	h2 := HashToken(input)
	if h1 != h2 {
		t.Error("HashToken is not deterministic")
	}
	hash := sha256.Sum256([]byte(input))
	expected := hex.EncodeToString(hash[:])
	if h1 != expected {
		t.Errorf("HashToken output does not match expected hash:\n  got:  %s\n  want: %s", h1, expected)
	}
}

func TestValidateSessionToken_Active(t *testing.T) {
	mq := &mockSessionQuerier{}
	s := NewSessionService(mq)

	rawToken, err := s.GenerateSessionToken(
		context.Background(),
		uuid.New(),
		"test",
		nil,
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}

	token, err := s.ValidateSessionToken(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("expected valid session token, got error: %v", err)
	}
	if token == nil {
		t.Fatal("expected non-nil session token")
	}
	if token.Status != repository.SessionTokenStatusActive {
		t.Errorf("expected status active, got %s", token.Status)
	}
}

func TestValidateSessionToken_Revoked(t *testing.T) {
	mq := &mockSessionQuerier{}
	s := NewSessionService(mq)

	rawToken, err := s.GenerateSessionToken(
		context.Background(),
		uuid.New(),
		"test",
		nil,
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}

	tokenHash := HashToken(rawToken)
	stored, err := mq.GetSessionTokenByHash(context.Background(), tokenHash)
	if err != nil {
		t.Fatal(err)
	}

	// Revoke the token
	_, err = mq.UpdateSessionTokenStatus(context.Background(), repository.UpdateSessionTokenStatusParams{
		ID:     stored.ID,
		Status: repository.SessionTokenStatusRevoked,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.ValidateSessionToken(context.Background(), rawToken)
	if err == nil {
		t.Fatal("expected error for revoked session token")
	}
}

func TestValidateSessionToken_Expired(t *testing.T) {
	mq := &mockSessionQuerier{}
	s := NewSessionService(mq)

	rawToken, err := s.GenerateSessionToken(
		context.Background(),
		uuid.New(),
		"test",
		nil,
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Manually expire the stored token by setting its ExpiresAt in the past.
	tokenHash := HashToken(rawToken)
	stored, err := mq.GetSessionTokenByHash(context.Background(), tokenHash)
	if err != nil {
		t.Fatal(err)
	}
	for userID, tokens := range mq.sessionsByUser {
		for i, t := range tokens {
			if t.ID == stored.ID {
				t.ExpiresAt = time.Now().UTC().Add(-time.Hour)
				mq.sessionsByUser[userID][i] = t
				break
			}
		}
	}

	_, err = s.ValidateSessionToken(context.Background(), rawToken)
	if err == nil {
		t.Fatal("expected error for expired session token")
	}
}

func TestValidateSessionToken_NotFound(t *testing.T) {
	s := NewSessionService(&mockSessionQuerier{})
	_, err := s.ValidateSessionToken(context.Background(), "st_nonexistent-token")
	if err == nil {
		t.Fatal("expected error for non-existent session token")
	}
}

func TestRevokeSessionToken_Success(t *testing.T) {
	mq := &mockSessionQuerier{}
	s := NewSessionService(mq)

	rawToken, err := s.GenerateSessionToken(
		context.Background(),
		uuid.New(),
		"test",
		nil,
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}

	tokenHash := HashToken(rawToken)
	stored, err := mq.GetSessionTokenByHash(context.Background(), tokenHash)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.RevokeSessionToken(context.Background(), stored.ID); err != nil {
		t.Fatalf("unexpected error revoking session token: %v", err)
	}

	// Verify it is now revoked
	updated, err := mq.GetSessionTokenByHash(context.Background(), tokenHash)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != repository.SessionTokenStatusRevoked {
		t.Errorf("expected status revoked, got %s", updated.Status)
	}
}

func TestGenerateSessionToken_WithScope(t *testing.T) {
	mq := &mockSessionQuerier{}
	s := NewSessionService(mq)
	scope := "read:usage"
	rawToken, err := s.GenerateSessionToken(
		context.Background(),
		uuid.New(),
		"scoped-test",
		&scope,
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rawToken, "st_") {
		t.Errorf("token should start with st_, got %q", rawToken)
	}

	// Verify the stored token has the correct scope.
	tokenHash := HashToken(rawToken)
	stored, err := mq.GetSessionTokenByHash(context.Background(), tokenHash)
	if err != nil {
		t.Fatal(err)
	}
	if !stored.Scope.Valid {
		t.Fatal("expected scope to be set")
	}
	if stored.Scope.String != scope {
		t.Errorf("expected scope %q, got %q", scope, stored.Scope.String)
	}
}

func TestRevokeSessionToken_NotFound(t *testing.T) {
	s := NewSessionService(&mockSessionQuerier{})
	err := s.RevokeSessionToken(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent session token")
	}
}

func TestRevokeSessionToken_AlreadyRevoked(t *testing.T) {
	mq := &mockSessionQuerier{}
	s := NewSessionService(mq)

	rawToken, err := s.GenerateSessionToken(
		context.Background(),
		uuid.New(),
		"test",
		nil,
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}

	tokenHash := HashToken(rawToken)
	stored, err := mq.GetSessionTokenByHash(context.Background(), tokenHash)
	if err != nil {
		t.Fatal(err)
	}

	// First revoke should succeed.
	if err := s.RevokeSessionToken(context.Background(), stored.ID); err != nil {
		t.Fatalf("unexpected error on first revoke: %v", err)
	}

	// Second revoke should fail (already revoked).
	if err := s.RevokeSessionToken(context.Background(), stored.ID); err == nil {
		t.Fatal("expected error when revoking already-revoked token")
	}
}
