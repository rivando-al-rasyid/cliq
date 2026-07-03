package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents the core identity record. Credentials only — no PII stored here.
// Username has been removed; identity is based on email.
type User struct {
	ID        uuid.UUID  `db:"id"`
	Email     string     `db:"email"`
	Password  string     `db:"password"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt *time.Time `db:"updated_at"`
}
