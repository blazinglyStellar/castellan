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
// and returns the raw token string.
func (s *SessionService) GenerateSessionToken(
	ctx context.Context,
	userID uuid.UUID,
	label string,
	scope *string,
	duration time.Duration,
) (rawToken string, err error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
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

	if _, err = s.queries.InsertSessionToken(ctx, params); err != nil {
		return "", fmt.Errorf("persist session token hash: %w", err)
	}

	return rawToken, nil
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

// RevokeSessionToken sets the session token's status to revoked.
// Only tokens with active status are revoked; already-revoked or expired
// tokens return an error.
func (s *SessionService) RevokeSessionToken(
	ctx context.Context,
	tokenID uuid.UUID,
) error {
	_, err := s.queries.RevokeSessionToken(ctx, tokenID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("session token not found or already revoked")
		}
		return fmt.Errorf("revoke session token: %w", err)
	}
	return nil
}

// HashToken returns the SHA-256 hex digest of the given token.
func HashToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}
