package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rivando-al-rasyid/vanwallet-backend/internals/model"
)

type ProfileRepo struct {
	db *pgxpool.Pool
}

func NewProfileRepo(db *pgxpool.Pool) *ProfileRepo {
	return &ProfileRepo{db: db}
}

func (p *ProfileRepo) UserProfile(ctx context.Context, email string) (model.Profile, error) {
	var profile model.Profile
	err := p.db.QueryRow(ctx, `
		SELECT p.full_name, p.phone, p.photo, p.created_at, p.updated_at
		FROM profiles p
		JOIN users u ON p.user_id = u.id
		WHERE u.email = $1`, email,
	).Scan(&profile.FullName, &profile.Phone, &profile.Photo, &profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Profile{}, errors.New("user profile not found")
		}
		return model.Profile{}, err
	}
	return profile, nil
}

func (p *ProfileRepo) EditProfile(ctx context.Context, email string, updates map[string]any) (model.Profile, error) {
	if len(updates) == 0 {
		return model.Profile{}, fmt.Errorf("EditProfile: no fields to update")
	}

	allowed := map[string]bool{
		"full_name": true,
		"phone":     true,
		"photo":     true,
	}

	var (
		sb      strings.Builder
		args    []any
		counter int
	)

	args = append(args, email)
	counter = 1

	sb.WriteString(`UPDATE profiles SET `)

	first := true
	for col, val := range updates {
		if !allowed[col] {
			return model.Profile{}, fmt.Errorf("EditProfile: column '%s' is not updatable", col)
		}
		if !first {
			sb.WriteString(", ")
		}
		counter++
		fmt.Fprintf(&sb, "%s = $%d", col, counter)
		args = append(args, val)
		first = false
	}

	sb.WriteString(`, updated_at = now()
        FROM users u
        WHERE profiles.user_id = u.id
          AND u.email = $1
        RETURNING
            profiles.user_id,
            profiles.full_name,
            profiles.phone,
            profiles.photo,
            profiles.updated_at`)

	var profile model.Profile
	err := p.db.QueryRow(ctx, sb.String(), args...).Scan(
		&profile.UserID,
		&profile.FullName,
		&profile.Phone,
		&profile.Photo,
		&profile.UpdatedAt,
	)
	if err != nil {
		return model.Profile{}, fmt.Errorf("EditProfile: %w", err)
	}
	return profile, nil
}

func (p *ProfileRepo) EditPin(ctx context.Context, email string, newPinHash string) (model.UserPin, error) {
	var userPin model.UserPin
	err := p.db.QueryRow(ctx, `
        UPDATE user_pins
        SET pin_hash = $2, updated_at = now()
        FROM users u
        WHERE user_pins.user_id = u.id
          AND u.email = $1
        RETURNING
            user_pins.pin_hash,
            user_pins.failed_attempts,
            user_pins.locked_until,
            user_pins.updated_at`, email, newPinHash,
	).Scan(&userPin.PinHash, &userPin.FailedAttempts, &userPin.LockedUntil, &userPin.UpdatedAt)
	if err != nil {
		return model.UserPin{}, fmt.Errorf("EditPin: %w", err)
	}
	return userPin, nil
}

func (p *ProfileRepo) GetCurrentPinHash(ctx context.Context, email string) (string, error) {
	var hash string
	err := p.db.QueryRow(ctx, `
		SELECT COALESCE(up.pin_hash, '')
		FROM user_pins up
		JOIN users u ON up.user_id = u.id
		WHERE u.email = $1`, email,
	).Scan(&hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil // no PIN row → treat as not set
		}
		return "", fmt.Errorf("GetCurrentPinHash: %w", err)
	}
	return hash, nil
}

// GetCurrentPassword returns the bcrypt hash of the user's current password.
func (p *ProfileRepo) GetCurrentPassword(ctx context.Context, email string) (string, error) {
	var hash string
	err := p.db.QueryRow(ctx, `SELECT password FROM users WHERE email = $1`, email).Scan(&hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("user not found")
		}
		return "", fmt.Errorf("GetCurrentPassword: %w", err)
	}
	return hash, nil
}

func (p *ProfileRepo) EditPassword(ctx context.Context, email string, newPassword string) (model.User, error) {
	var user model.User
	err := p.db.QueryRow(ctx, `
        UPDATE users
        SET password = $2, updated_at = NOW()
        WHERE email = $1
        RETURNING password, updated_at`, email, newPassword,
	).Scan(&user.Password, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, errors.New("user not found")
		}
		return model.User{}, fmt.Errorf("EditPassword: %w", err)
	}
	return user, nil
}

// GetUserInfo returns profile + total wallet balance in a single query.
// Used for the app header (avatar, name, balance).
func (p *ProfileRepo) GetUserInfo(ctx context.Context, email string) (model.Profile, int64, error) {
	var profile model.Profile
	var balance int64

	err := p.db.QueryRow(ctx, `
        SELECT
            p.full_name,
            p.phone,
            p.photo,
            COALESCE(SUM(w.balance), 0) AS current_balance,
            COALESCE(up.pin_hash, '') AS pin_hash,
            COALESCE((ARRAY_AGG(w.id ORDER BY w.created_at ASC))[1]) AS wallet_id
        FROM profiles p
        JOIN users u ON p.user_id = u.id
        LEFT JOIN user_pins up ON p.user_id = up.user_id
        LEFT JOIN wallets w ON w.user_id = u.id
        WHERE u.email = $1
        GROUP BY p.full_name, p.phone, p.photo, up.pin_hash`, email,
	).Scan(&profile.FullName, &profile.Phone, &profile.Photo, &balance, &profile.PinHash, &profile.WalletID)

	if err != nil {
		return model.Profile{}, 0, err
	}

	return profile, balance, nil
}
