package dto

// TopupRequest is the payload for initiating a wallet top-up.
// Pin is verified server-side before the record is created.
type TopupRequest struct {
	WalletID      string `json:"wallet_id"      binding:"required"`
	Amount        int64  `json:"amount"         binding:"required,gt=0"`
	PaymentMethod string `json:"payment_method" binding:"required,oneof=BRI BCA DANA GOPAY OVO"`
}

// TopupResponse is returned after a top-up is initiated.
type TopupResponse struct {
	ID                string `json:"id"`
	WalletID          string `json:"wallet_id"`
	Amount            int64  `json:"amount"`
	PaymentMethod     string `json:"payment_method"`
	ExternalReference string `json:"external_reference,omitempty"`
	Status            string `json:"status"`
	CreatedAt         string `json:"created_at"`
}
