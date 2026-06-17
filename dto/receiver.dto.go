package dto

// ReceiverResult is a single search result for a potential transfer recipient.
// Search is performed against name, email, phone, wallet label, and wallet id.
type ReceiverResult struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	FullName    string `json:"full_name,omitempty"`
	Phone       string `json:"phone,omitempty"`
	Photo       string `json:"photo,omitempty"`
	WalletID    string `json:"wallet_id"`
	WalletLabel string `json:"wallet_label"`
}

// ReceiverListResponse wraps a paginated receiver search result.
type ReceiverListResponse struct {
	Data       []ReceiverResult `json:"data"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	Limit      int              `json:"limit"`
	TotalPages int              `json:"total_pages"`
}
