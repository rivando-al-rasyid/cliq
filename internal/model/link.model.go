package model

import (
	"time"

	"github.com/google/uuid"
)

type Link struct {
	ID         uuid.UUID `db:"id"`
	UserID     uuid.UUID `db:"user_id"`
	OriginLink string    `db:"origin_link"`
	Slug       string    `db:"slug"`
	CreatedAt  time.Time `db:"created_at"`
}
