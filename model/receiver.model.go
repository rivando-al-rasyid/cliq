package model

import "github.com/google/uuid"

// ReceiverResult is the projection used when searching for a transfer recipient.
// Search is done by full_name or phone (profiles table).
type ReceiverResult struct {
	UserID      uuid.UUID `db:"user_id"`
	Email       string    `db:"email"`
	FullName    *string   `db:"full_name"`
	Phone       *string   `db:"phone"`
	Photo       *string   `db:"photo"`
	WalletID    uuid.UUID `db:"wallet_id"`
	WalletLabel string    `db:"wallet_label"`
}
