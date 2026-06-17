package model

import (
	"time"

	"github.com/google/uuid"
)

// Transfer links BOTH sides of a peer-to-peer transfer.
//   - TransactionID          → the TRANSFER_OUT transaction row (sender's ledger entry)
//   - RecipientTransactionID → the TRANSFER_IN  transaction row (recipient's ledger entry)
//   - TransferCode           → optional human-readable reference code
type Transfer struct {
	TransactionID          uuid.UUID  `db:"transaction_id"`
	RecipientTransactionID uuid.UUID  `db:"recipient_transaction_id"`
	TransferCode           *string    `db:"transfer_code"`
	CreatedAt              time.Time  `db:"created_at"`
}
