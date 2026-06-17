package model

import (
	"time"

	"github.com/google/uuid"
)

// Topup is a standalone top-up record (NOT part of the transactions ledger).
// ExternalReference uniqueness enables payment-gateway idempotency checks.
type Topup struct {
	ID                uuid.UUID         `db:"id"`
	WalletID          uuid.UUID         `db:"wallet_id"`
	Amount            int64             `db:"amount"`
	Status            TransactionStatus `db:"status"`
	PaymentMethod     *PaymentMethod    `db:"payment_method"`
	PaymentMetadata   *[]byte           `db:"payment_metadata"` // jsonb
	ExternalReference *string           `db:"external_reference"`
	CreatedAt         time.Time         `db:"created_at"`
	UpdatedAt         *time.Time        `db:"updated_at"`
}
