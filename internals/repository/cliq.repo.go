package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CliqRepo struct {
	db *pgxpool.Pool
}

func NewCliqRepo(db *pgxpool.Pool) *CliqRepo {
	return &CliqRepo{db: db}
}

func (c *CliqRepo) CreateSlug(
	ctx context.Context,
	userID uuid.UUID,
	originLink string,
	slug string,
) error {
	_, err := c.db.Exec(ctx,
		`
		INSERT INTO links (
			user_id,
			origin_link,
			slug
		)
		VALUES ($1, $2, $3)
		`,
		userID,
		originLink,
		slug,
	)
	if err != nil {
		return fmt.Errorf("create slug: %w", err)
	}

	return nil
}

func (c *CliqRepo) GetOriginLinkBySlug(ctx context.Context, slug string) (string, error) {
	var originLink string

	err := c.db.QueryRow(ctx,
		`
		SELECT origin_link
		FROM links
		WHERE slug = $1
		  AND is_deleted = false
		LIMIT 1
		`,
		slug,
	).Scan(&originLink)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", pgx.ErrNoRows
		}

		return "", fmt.Errorf("get origin link by slug: %w", err)
	}

	return originLink, nil
}
