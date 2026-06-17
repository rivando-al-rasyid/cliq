package dto

// SummaryResponse returns the user's financial overview for the dashboard.
type SummaryResponse struct {
	CurrentBalance int64        `json:"current_balance"`
	TotalIncome    int64        `json:"total_income"`
	TotalExpense   int64        `json:"total_expense"`
	Wallets        []WalletItem `json:"wallets"`
}

// WalletItem is a condensed wallet entry inside SummaryResponse.
type WalletItem struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Balance int64  `json:"balance"`
}

// ChartPointResponse is a single bar in the financial chart.
type ChartPointResponse struct {
	Label   string `json:"label"`
	Income  int64  `json:"income,omitempty"`
	Expense int64  `json:"expense,omitempty"`
}

// TransactionReportResponse wraps the full chart dataset.
// Type: "income" | "expense" | "both"
// Range: "7days" | "30days"
type TransactionReportResponse struct {
	Range  string               `json:"range"`
	Type   string               `json:"type"`
	Points []ChartPointResponse `json:"points"`
}

// TransactionResponse is the public representation of a single transaction ledger entry.
type TransactionResponse struct {
	ID             string `json:"id"`
	WalletID       string `json:"wallet_id"`
	Type           string `json:"type"`
	Amount         int64  `json:"amount"`
	AdminFee       int64  `json:"admin_fee"`
	Status         string `json:"status"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	Note           string `json:"note,omitempty"`
	CreatedAt      string `json:"created_at"`
}

// TransactionListResponse wraps a paginated list of transactions.
type TransactionListResponse struct {
	Data  []TransactionResponse `json:"data"`
	Total int                   `json:"total"`
	Page  int                   `json:"page"`
	Limit int                   `json:"limit"`
}
