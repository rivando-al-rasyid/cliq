package dto

import "github.com/google/uuid"

// RegisterRequest — only email and password required.
// No username field; username concept has been removed entirely.
type RegisterRequest struct {
	Email    string `json:"email"    binding:"required,email"  example:"user@example.com"`
	Password string `json:"password" binding:"required,min=8"  example:"P@ssw0rd123" minLength:"8"`
}

// LoginRequest is the payload for authenticating.
type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required"       example:"P@ssw0rd123"`
}

type ResetPasswordRequest struct {
	Email string `json:"email" binding:"required,email" example:"user@example.com"`
}

// ConfirmResetPassword is the payload for POST /auth/reset/confirm.
// Only the opaque token is needed — the user is identified via the token's DB record.
type ConfirmResetPassword struct {
	Token string `json:"token" binding:"required" example:"abc123xyz"`
}

// ChangePasswordRequest is the payload for POST /auth/change-password.
// Requires a valid password-reset JWT (sub="password-reset") in the Authorization header.
type ChangeAndPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=8" example:"NewP@ssw0rd!" minLength:"8"`
}

// UserResponse is the public representation after register/login.
type UserResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}
