package model

import (
	"time"

	"github.com/google/uuid"
)

// UserPin stores the bcrypt-hashed PIN for a user.
// LockedUntil enables brute-force protection — nil means not locked.
type UserPin struct {
	ID             uuid.UUID  `db:"id"`
	UserID         uuid.UUID  `db:"user_id"`
	PinHash        *string    `db:"pin_hash"`
	FailedAttempts int        `db:"failed_attempts"`
	LockedUntil    *time.Time `db:"locked_until"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      *time.Time `db:"updated_at"`
}
