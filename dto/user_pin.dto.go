package dto

// SetPinRequest is the payload for setting or updating a user's PIN.

type SetPinRequest struct {
	OldPin  string  `json:"old_pin"`
	PinHash *string `json:"pin_hash" binding:"required,len=6,numeric"`
}

// VerifyPinRequest is the payload for verifying a user's PIN.
type VerifyPinRequest struct {
	Pin string `json:"pin" binding:"required,len=6,numeric"`
}
