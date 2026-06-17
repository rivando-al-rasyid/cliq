package dto

// HistoryItem is a unified history entry covering both top-ups and ledger transactions.
// Source: "topup" | "transaction"
type HistoryItem struct {
	ID            string `json:"id"`
	Source        string `json:"source"`    // "topup" | "transaction"
	Type          string `json:"type"`      // TransactionType or "TOPUP"
	Direction     string `json:"direction"` // "income" | "expense"
	Title         string `json:"title"`
	Amount        int64  `json:"amount"`
	AdminFee      int64  `json:"admin_fee,omitempty"`
	Status        string `json:"status"`
	PaymentMethod string `json:"payment_method,omitempty"` // only for topups
	Note          string `json:"note,omitempty"`
	WalletID      string `json:"wallet_id"`
	WalletLabel   string `json:"wallet_label"`
	CreatedAt     string `json:"created_at"`
}

// HistoryListResponse wraps a paginated unified history list.
type HistoryListResponse struct {
	Data       []HistoryItem `json:"data"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
	TotalPages int           `json:"total_pages"`
}
