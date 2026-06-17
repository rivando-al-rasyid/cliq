package model

import "github.com/google/uuid"

// Withdrawal holds bank account details for a manual bank withdrawal.
// BankName, AccountNumber, and AccountHolder are snapshot values captured
// at the time of withdrawal — not foreign keys to a bank account table.
type Withdrawal struct {
	TransactionID uuid.UUID `db:"transaction_id"`
	BankName      string    `db:"bank_name"`
	AccountNumber string    `db:"account_number"`
	AccountHolder string    `db:"account_holder"`
}
