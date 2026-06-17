package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rivando-al-rasyid/cliq/internals/dto"
)

var ErrLinkNotFound = errors.New("link not found")

type CliqRepository interface {
	CreateSlug(ctx context.Context, userID uuid.UUID, originLink string, slug string) error
	GetOriginLinkBySlug(ctx context.Context, slug string) (string, error)
}

type CliqService struct {
	repo CliqRepository
	rdb  *redis.Client
}

func NewCliqService(repo CliqRepository, rdb *redis.Client) *CliqService {
	return &CliqService{repo: repo, rdb: rdb}
}

func (c *CliqService) CreateSlug(ctx context.Context, userID uuid.UUID, link dto.Link) (string, error) {
	if err := c.repo.CreateSlug(ctx, userID, link.OriginLink, link.Slug); err != nil {
		return "", err
	}

	return link.Slug, nil
}

func (c *CliqService) RedirectBySlug(ctx context.Context, slug string) (string, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return "", ErrLinkNotFound
	}

	originLink, err := c.repo.GetOriginLinkBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrLinkNotFound
		}

		return "", err
	}

	return originLink, nil
}
