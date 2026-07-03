package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rivando-al-rasyid/cliq-backend/internals/model"
)

type LinkRepo struct {
	db *pgxpool.Pool
}

func NewLinkRepo(db *pgxpool.Pool) *LinkRepo {
	return &LinkRepo{db: db}
}

func (c *LinkRepo) CreateSlug(
	ctx context.Context,
	userID uuid.UUID,
	originLink string,
	slug string,
) (model.Link, error) {
	var link model.Link

	err := c.db.QueryRow(ctx,
		`
		INSERT INTO links (
			user_id,
			origin_link,
			slug
		)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, origin_link, slug, clicks, created_at
		`,
		userID,
		originLink,
		slug,
	).Scan(&link.ID, &link.UserID, &link.OriginLink, &link.Slug, &link.Clicks, &link.CreatedAt)
	if err != nil {
		return model.Link{}, fmt.Errorf("create slug: %w", err)
	}

	return link, nil
}

func (c *LinkRepo) GetOriginLinkBySlug(ctx context.Context, slug string) (string, error) {
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

func (c *LinkRepo) GetSlugByID(ctx context.Context, userID uuid.UUID, linkID uuid.UUID) (string, error) {
	var slug string

	err := c.db.QueryRow(ctx,
		`
		SELECT slug
		FROM links
		WHERE id = $1
		  AND user_id = $2
		  AND is_deleted = false
		LIMIT 1
		`,
		linkID,
		userID,
	).Scan(&slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", pgx.ErrNoRows
		}

		return "", fmt.Errorf("get slug by id: %w", err)
	}

	return slug, nil
}

func (c *LinkRepo) ListLinksByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Link, int, error) {
	var total int
	if err := c.db.QueryRow(ctx,
		`
		SELECT COUNT(*)
		FROM links
		WHERE user_id = $1
		  AND is_deleted = false
		`,
		userID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count links by user: %w", err)
	}

	rows, err := c.db.Query(ctx,
		`
		SELECT id, user_id, origin_link, slug, clicks, created_at
		FROM links
		WHERE user_id = $1
		  AND is_deleted = false
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
		`,
		userID,
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list links by user: %w", err)
	}
	defer rows.Close()

	links := make([]model.Link, 0)
	for rows.Next() {
		var link model.Link
		if err := rows.Scan(&link.ID, &link.UserID, &link.OriginLink, &link.Slug, &link.Clicks, &link.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan link: %w", err)
		}
		links = append(links, link)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate links: %w", err)
	}

	return links, total, nil
}

func (c *LinkRepo) ListActiveSlugsByUser(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := c.db.Query(ctx,
		`
		SELECT slug
		FROM links
		WHERE user_id = $1
		  AND is_deleted = false
		`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list active slugs by user: %w", err)
	}
	defer rows.Close()

	slugs := make([]string, 0)
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, fmt.Errorf("scan active slug: %w", err)
		}
		slugs = append(slugs, slug)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active slugs: %w", err)
	}

	return slugs, nil
}

func (c *LinkRepo) GetTotalClicksByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var totalClicks int

	err := c.db.QueryRow(ctx,
		`
		SELECT COALESCE(SUM(clicks), 0)
		FROM links
		WHERE user_id = $1
		  AND is_deleted = false
		`,
		userID,
	).Scan(&totalClicks)
	if err != nil {
		return 0, fmt.Errorf("get total clicks by user: %w", err)
	}

	return totalClicks, nil
}

func (c *LinkRepo) IncrementClicksBySlug(ctx context.Context, slug string, delta int64) error {
	if delta <= 0 {
		return nil
	}

	result, err := c.db.Exec(ctx,
		`
		UPDATE links
		SET clicks = clicks + $2,
		    updated_at = now()
		WHERE slug = $1
		`,
		slug,
		delta,
	)
	if err != nil {
		return fmt.Errorf("increment clicks by slug: %w", err)
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (c *LinkRepo) SoftDeleteLinkByID(ctx context.Context, userID uuid.UUID, linkID uuid.UUID) (string, error) {
	var slug string

	err := c.db.QueryRow(ctx,
		`
		UPDATE links
		SET is_deleted = true,
		    updated_at = now(),
		    deleted_at = now()
		WHERE id = $1
		  AND user_id = $2
		  AND is_deleted = false
		RETURNING slug
		`,
		linkID,
		userID,
	).Scan(&slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", pgx.ErrNoRows
		}

		return "", fmt.Errorf("soft delete link by id: %w", err)
	}

	return slug, nil
}
