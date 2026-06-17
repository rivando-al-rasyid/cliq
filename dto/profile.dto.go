package dto

import "mime/multipart"

// UpdateProfileRequest is the payload for updating a user's profile (multipart form).
type UpdateProfileRequest struct {
	FullName *string               `form:"full_name" binding:"omitempty,min=2"`
	Phone    *string               `form:"phone"     binding:"omitempty,e164"`
	Photo    *multipart.FileHeader `form:"photo"     binding:"omitempty"`
}

// ProfileResponse is the public representation of a user profile.
type ProfileResponse struct {
	FullName *string `json:"full_name,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Photo    *string `json:"photo,omitempty"`
}

// ChangePasswordRequest is the payload for updating the authenticated user's password.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	Password    string `json:"password"     binding:"required,min=8"`
}
