package model

import (
	"time"

	"github.com/google/uuid"
)

// TokenType maps to the token_type DB enum.
type TokenType string

const (
	TokenTypeAccess            TokenType = "ACCESS"
	TokenTypePasswordReset     TokenType = "PASSWORD_RESET"
	TokenTypeEmailVerification TokenType = "EMAIL_VERIFICATION"
)

// Token represents a row in the tokens table.
type Token struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	Token     string    `db:"token"`
	Type      TokenType `db:"type"`
	ExpiresAt time.Time `db:"expires_at"`
	IsRevoked bool      `db:"is_revoked"`
	CreatedAt time.Time `db:"created_at"`
}
