package dto

// TransferRequest is the payload for transferring funds to another wallet.
// Pin is verified server-side before the transfer is committed.
type TransferRequest struct {
	SenderWalletID    string `json:"sender_wallet_id"    binding:"required"`
	RecipientWalletID string `json:"recipient_wallet_id" binding:"required"`
	Amount            int64  `json:"amount"              binding:"required,gt=0"`
	Note              string `json:"note"                binding:"omitempty,max=255"`
	Pin               string `json:"pin"                 binding:"required,len=6"`
}

// TransferResponse is returned to the SENDER only.
// The recipient receives an in-app/push notification; their transaction
// detail is NOT exposed in this response.
type TransferResponse struct {
	TransferCode  string              `json:"transfer_code"`
	SenderTx      TransactionResponse `json:"sender_transaction"`
	RecipientInfo RecipientInfo       `json:"recipient_info"`
}

// RecipientInfo carries the minimal info shown to the sender about the destination.
type RecipientInfo struct {
	WalletID string `json:"wallet_id"`
	Amount   int64  `json:"amount"`
}
