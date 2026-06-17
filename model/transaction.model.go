package model

import (
	"time"

	"github.com/google/uuid"
)

// WalletSummary is a lightweight wallet projection used inside TransactionSummary.
type WalletSummary struct {
	ID      uuid.UUID
	Label   string
	Balance int64
}

// TransactionSummary holds aggregated financial data for a user.
type TransactionSummary struct {
	CurrentBalance int64
	TotalIncome    int64
	TotalExpense   int64
	Wallets        []WalletSummary
}

// ChartPoint is a single data point in the income/expense report chart.
type ChartPoint struct {
	Label   string
	Income  int64
	Expense int64
}

// Transaction maps to the central ledger table.
// Type matches the DB enum: EXPENSE, WITHDRAWAL, TRANSFER_IN, TRANSFER_OUT.
// Amount is always positive; direction is inferred from Type.
type Transaction struct {
	ID             uuid.UUID         `db:"id"`
	WalletID       uuid.UUID         `db:"wallet_id"`
	Type           TransactionType   `db:"type"`
	Amount         int64             `db:"amount"`
	AdminFee       int64             `db:"admin_fee"`
	Status         TransactionStatus `db:"status"`
	IdempotencyKey *string           `db:"idempotency_key"`
	Note           *string           `db:"note"`
	CreatedAt      time.Time         `db:"created_at"`
	UpdatedAt      *time.Time        `db:"updated_at"`
}
