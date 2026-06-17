package model

// TransactionStatus represents the status of a transaction.
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "PENDING"
	TransactionStatusSuccess   TransactionStatus = "SUCCESS"
	TransactionStatusFailed    TransactionStatus = "FAILED"
	TransactionStatusCancelled TransactionStatus = "CANCELLED"
)

// PaymentMethod represents supported top-up payment methods.
type PaymentMethod string

const (
	PaymentMethodBRI   PaymentMethod = "BRI"
	PaymentMethodBCA   PaymentMethod = "BCA"
	PaymentMethodDANA  PaymentMethod = "DANA"
	PaymentMethodGoPay PaymentMethod = "GOPAY"
	PaymentMethodOVO   PaymentMethod = "OVO"
)

// TransactionType represents the type of a transaction (matches DB enum).
type TransactionType string

const (
	TransactionTypeExpense      TransactionType = "EXPENSE"
	TransactionTypeWithdrawal   TransactionType = "WITHDRAWAL"
	TransactionTypeTransferIn   TransactionType = "TRANSFER_IN"
	TransactionTypeTransferOut  TransactionType = "TRANSFER_OUT"
)
