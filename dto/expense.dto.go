package dto

// ExpenseRequest is the payload for recording an expense.
// Pin is verified server-side before the expense is committed.
type ExpenseRequest struct {
	WalletID     string  `json:"wallet_id"     binding:"required"`
	Amount       int64   `json:"amount"        binding:"required,gt=0"`
	AdminFee     int64   `json:"admin_fee"     binding:"omitempty,gte=0"`
	Category     string  `json:"category"      binding:"omitempty,max=50"`
	MerchantName string  `json:"merchant_name" binding:"omitempty,max=100"`
	Note         *string `json:"note"          binding:"omitempty,max=255"`
	Pin          string  `json:"pin"           binding:"required,len=6"`
}

// ExpenseResponse is returned after an expense is recorded.
type ExpenseResponse struct {
	TransactionID string `json:"transaction_id"`
	WalletID      string `json:"wallet_id"`
	Amount        int64  `json:"amount"`
	AdminFee      int64  `json:"admin_fee"`
	Category      string `json:"category,omitempty"`
	MerchantName  string `json:"merchant_name,omitempty"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}
