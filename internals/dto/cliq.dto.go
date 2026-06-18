package dto

import "time"

// Link is the create-link payload.
// Slug is optional. If it is empty, the backend generates a random slug.
type Link struct {
	OriginLink string `json:"origin_link" binding:"required,url"`
	Slug       string `json:"slug,omitempty" binding:"omitempty,min=3,max=50"`
}

type LinkResponse struct {
	ID         string    `json:"id,omitempty"`
	OriginLink string    `json:"origin_link"`
	Slug       string    `json:"slug"`
	ShortURL   string    `json:"short_url"`
	Clicks     int       `json:"clicks"`
	CreatedAt  time.Time `json:"created_at"`
}

type DashboardResponse struct {
	Links       []LinkResponse `json:"links"`
	TotalActive int            `json:"total_active"`
	TotalClicks int            `json:"total_clicks"`
	Page        int            `json:"page"`
	Limit       int            `json:"limit"`
	TotalPages  int            `json:"total_pages"`
}
