package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rivando-al-rasyid/cliq/internals/model"
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
		RETURNING id, user_id, origin_link, slug, created_at
		`,
		userID,
		originLink,
		slug,
	).Scan(&link.ID, &link.UserID, &link.OriginLink, &link.Slug, &link.CreatedAt)
	if err != nil {
		return model.Link{}, fmt.Errorf("create slug: %w", err)
	}

	return link, nil
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

func (c *CliqRepo) ListLinksByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Link, int, error) {
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
		SELECT id, user_id, origin_link, slug, created_at
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
		if err := rows.Scan(&link.ID, &link.UserID, &link.OriginLink, &link.Slug, &link.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan link: %w", err)
		}
		links = append(links, link)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate links: %w", err)
	}

	return links, total, nil
}

func (c *CliqRepo) SoftDeleteLinkByID(ctx context.Context, userID uuid.UUID, linkID uuid.UUID) error {
	result, err := c.db.Exec(ctx,
		`
		UPDATE links
		SET is_deleted = true
		WHERE id = $1
		  AND user_id = $2
		  AND is_deleted = false
		`,
		linkID,
		userID,
	)
	if err != nil {
		return fmt.Errorf("soft delete link by id: %w", err)
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}
