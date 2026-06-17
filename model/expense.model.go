package model

import "github.com/google/uuid"

// Expense is auto-created by the system for every OUT transaction that is
// NOT a withdrawal. Category and MerchantName can be enriched later.
type Expense struct {
	TransactionID uuid.UUID `db:"transaction_id"`
	Category      *string   `db:"category"`
	MerchantName  *string   `db:"merchant_name"`
}
