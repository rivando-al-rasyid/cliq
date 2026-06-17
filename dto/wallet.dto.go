package dto

// CreateWalletRequest is the payload for creating a new wallet.
type CreateWalletRequest struct {
	Label string `json:"label" binding:"omitempty,min=1,max=50"`
}

// UpdateWalletRequest is the payload for updating a wallet's label.
type UpdateWalletRequest struct {
	Label string `json:"label" binding:"required,min=1,max=50"`
}

// WalletResponse is the public representation of a wallet.
type WalletResponse struct {
	ID      string `json:"id"`
	UserID  string `json:"user_id"`
	Label   string `json:"label"`
	Balance int64  `json:"balance"`
}
