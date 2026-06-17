package model

import (
	"time"

	"github.com/google/uuid"
)

// Wallet represents a user's wallet.
// Supports multiple wallets per user.
// Balance is stored in the smallest currency unit (sen/IDR).
type Wallet struct {
	ID        uuid.UUID  `db:"id"`
	UserID    uuid.UUID  `db:"user_id"`
	Label     string     `db:"label"`
	Balance   int64      `db:"balance"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt *time.Time `db:"updated_at"`
}
