package model

import "time"

// HistoryItem is a flat unified record used by GetAllHistory.
// Source is "topup" or "transaction".
type HistoryItem struct {
	ID            string
	Source        string
	Type          string
	Direction     string
	Amount        int64
	AdminFee      int64
	Status        string
	PaymentMethod string
	Note          string
	WalletID      string
	WalletLabel   string
	CreatedAt     time.Time
}

// HistoryFilter contains optional filters for the unified transaction history feed.
// Direction accepts income or expense. Query is used for simple search over text fields.
type HistoryFilter struct {
	Page      int
	Limit     int
	WalletID  string
	Source    string
	Type      string
	Status    string
	Direction string
	StartDate string
	EndDate   string
	Query     string
}
