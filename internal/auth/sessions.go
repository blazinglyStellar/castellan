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

// ListSessionsItem is a single session token returned in list responses.
// It omits the token_hash and user_id for security.
type ListSessionsItem struct {
	ID        uuid.UUID                     `json:"id"`
	Label     pgtype.Text                   `json:"label"`
	Scope     pgtype.Text                   `json:"scope"`
	Status    repository.SessionTokenStatus `json:"status"`
	ExpiresAt time.Time                     `json:"expires_at"`
	CreatedAt time.Time                     `json:"created_at"`
}

var (
	ErrSessionTokenNotFound  = errors.New("session token not found")
	ErrSessionTokenNotActive = errors.New("session token is not active")
)

const tokenBytes = 32

// SessionService generates, validates, and revokes session tokens.
type SessionService struct {
	queries repository.Querier
}

// NewSessionService returns a SessionService backed by the given repository.
func NewSessionService(queries repository.Querier) *SessionService {
	return &SessionService{queries: queries}
}

// GenerateSessionToken creates a new session token (st_ prefix, 32 random
// bytes, URL-safe base64 without padding), persists only its SHA-256 hash,
// and returns the raw token string and the persisted record.
func (s *SessionService) GenerateSessionToken(
	ctx context.Context,
	userID uuid.UUID,
	label string,
	scope *string,
	duration time.Duration,
) (rawToken string, token repository.SessionToken, err error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", repository.SessionToken{}, fmt.Errorf("generate random bytes: %w", err)
	}

	rawToken = "st_" + base64.RawURLEncoding.EncodeToString(raw)
	tokenHash := HashToken(rawToken)

	var scopeVal pgtype.Text
	if scope != nil {
		scopeVal = pgtype.Text{String: *scope, Valid: true}
	}

	now := time.Now().UTC()
	params := repository.InsertSessionTokenParams{
		UserID:    userID,
		TokenHash: tokenHash,
		Label:     pgtype.Text{String: label, Valid: label != ""},
		Scope:     scopeVal,
		Status:    repository.SessionTokenStatusActive,
		ExpiresAt: now.Add(duration),
		CreatedAt: now,
	}

	token, err = s.queries.InsertSessionToken(ctx, params)
	if err != nil {
		return "", repository.SessionToken{}, fmt.Errorf("persist session token hash: %w", err)
	}

	return rawToken, token, nil
}

// ValidateSessionToken hashes the raw token, looks up the record, and verifies
// it is active and not expired. Returns the session token record on success.
func (s *SessionService) ValidateSessionToken(
	ctx context.Context,
	rawToken string,
) (*repository.SessionToken, error) {
	tokenHash := HashToken(rawToken)

	token, err := s.queries.GetSessionTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("session token not found")
		}
		return nil, fmt.Errorf("lookup session token: %w", err)
	}

	if token.Status != repository.SessionTokenStatusActive {
		return nil, errors.New("session token is not active")
	}

	if token.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("session token has expired")
	}

	return &token, nil
}

// GetSessionByID retrieves a session token by ID, verifying it belongs to the
// given user. Returns ErrSessionTokenNotFound if not found.
func (s *SessionService) GetSessionByID(ctx context.Context, tokenID, userID uuid.UUID) (*repository.SessionToken, error) {
	tokens, err := s.queries.ListSessionTokensByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("lookup session token: %w", err)
	}
	for _, t := range tokens {
		if t.ID == tokenID {
			return &t, nil
		}
	}
	return nil, ErrSessionTokenNotFound
}

// RevokeSessionToken sets the session token's status to revoked.
// Only tokens with active status are revoked; already-revoked or expired
// tokens return an error.
func (s *SessionService) RevokeSessionToken(
	ctx context.Context,
	tokenID, userID uuid.UUID,
) (*repository.SessionToken, error) {
	token, err := s.GetSessionByID(ctx, tokenID, userID)
	if err != nil {
		return nil, err
	}
	if token.Status != repository.SessionTokenStatusActive {
		return nil, ErrSessionTokenNotActive
	}

	updated, err := s.queries.RevokeSessionToken(ctx, tokenID)
	if err != nil {
		return nil, fmt.Errorf("revoke session token: %w", err)
	}
	return &updated, nil
}

// ListSessions returns all session tokens for the given user (metadata only).
// Token hashes are never exposed.
func (s *SessionService) ListSessions(ctx context.Context, userID uuid.UUID) ([]ListSessionsItem, error) {
	tokens, err := s.queries.ListSessionTokensByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	items := make([]ListSessionsItem, 0, len(tokens))
	for _, t := range tokens {
		items = append(items, ListSessionsItem{
			ID:        t.ID,
			Label:     t.Label,
			Scope:     t.Scope,
			Status:    t.Status,
			ExpiresAt: t.ExpiresAt,
			CreatedAt: t.CreatedAt,
		})
	}

	return items, nil
}

// HashToken returns the SHA-256 hex digest of the given token.
func HashToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}
