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

var (
	ErrSessionNotFound  = errors.New("session not found")
	ErrSessionNotActive = errors.New("session is not active")
	ErrSessionExpired   = errors.New("session has expired")
)

const (
	sessionTokenBytes = 32
	defaultSessionTTL = 7 * 24 * time.Hour
)

type SessionService struct {
	queries repository.Querier
}

func NewSessionService(queries repository.Querier) *SessionService {
	return &SessionService{queries: queries}
}

func (s *SessionService) CreateSession(ctx context.Context, userID uuid.UUID, duration time.Duration) (rawToken string, err error) {
	if duration <= 0 {
		duration = defaultSessionTTL
	}

	raw := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}

	rawToken = base64.RawURLEncoding.EncodeToString(raw)
	tokenHash := HashToken(rawToken)

	now := time.Now().UTC()
	params := repository.InsertSessionTokenParams{
		UserID:    userID,
		TokenHash: tokenHash,
		Label:     pgtype.Text{String: "oauth-session", Valid: true},
		Status:    repository.SessionTokenStatusActive,
		ExpiresAt: now.Add(duration),
		CreatedAt: now,
	}

	if _, err := s.queries.InsertSessionToken(ctx, params); err != nil {
		return "", fmt.Errorf("persist session hash: %w", err)
	}

	return rawToken, nil
}

func (s *SessionService) ValidateSession(ctx context.Context, rawToken string) (*repository.SessionToken, error) {
	tokenHash := HashToken(rawToken)

	token, err := s.queries.GetSessionTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("lookup session: %w", err)
	}

	if token.Status != repository.SessionTokenStatusActive {
		return nil, ErrSessionNotActive
	}

	if !token.ExpiresAt.After(time.Now()) {
		return nil, ErrSessionExpired
	}

	return &token, nil
}

func (s *SessionService) RevokeSession(ctx context.Context, rawToken string) error {
	tokenHash := HashToken(rawToken)

	token, err := s.queries.GetSessionTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("lookup session: %w", err)
	}

	if token.Status != repository.SessionTokenStatusActive {
		return ErrSessionNotActive
	}

	if _, err := s.queries.RevokeSessionToken(ctx, token.ID); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}

	return nil
}

func HashToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}
