package service

import (
	"context"
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rivando-al-rasyid/cliq/internals/dto"
	"github.com/rivando-al-rasyid/cliq/internals/model"
)

var (
	ErrLinkNotFound      = errors.New("link not found")
	ErrInvalidOriginLink = errors.New("origin link must start with http:// or https://")
	ErrInvalidSlug       = errors.New("slug must be 3-50 characters and can only contain letters, numbers, and hyphens")
	ErrReservedSlug      = errors.New("slug is reserved and cannot be used")
	ErrSlugAlreadyExists = errors.New("slug already exists")
	ErrInvalidLinkID     = errors.New("invalid link id")
	validSlugPattern     = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
	reservedSlugs        = map[string]struct{}{
		"api":       {},
		"login":     {},
		"register":  {},
		"dashboard": {},

		// Internal backend route prefixes. These are stricter than the spec,
		// but they prevent public slugs from colliding with app/API routes.
		"auth":    {},
		"link":    {},
		"profile": {},
		"swagger": {},
		"img":     {},
	}
)

const (
	autoSlugLength = 6
	maxSlugRetries = 10
	slugAlphabet   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type CliqRepository interface {
	CreateSlug(ctx context.Context, userID uuid.UUID, originLink string, slug string) (model.Link, error)
	GetOriginLinkBySlug(ctx context.Context, slug string) (string, error)
	ListLinksByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Link, int, error)
	SoftDeleteLinkByID(ctx context.Context, userID uuid.UUID, linkID uuid.UUID) error
}

type CliqService struct {
	repo CliqRepository
	rdb  *redis.Client
}

func NewCliqService(repo CliqRepository, rdb *redis.Client) *CliqService {
	return &CliqService{repo: repo, rdb: rdb}
}

func normalizeSlug(slug string) string {
	return strings.TrimSpace(strings.ReplaceAll(slug, " ", "-"))
}

func validateOriginLink(originLink string) error {
	parsed, err := url.ParseRequestURI(originLink)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ErrInvalidOriginLink
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidOriginLink
	}

	return nil
}

func validateSlug(slug string) error {
	if slug == "" || len(slug) < 3 || len(slug) > 50 || !validSlugPattern.MatchString(slug) {
		return ErrInvalidSlug
	}

	if _, reserved := reservedSlugs[strings.ToLower(slug)]; reserved {
		return ErrReservedSlug
	}

	return nil
}

func randomSlug(length int) (string, error) {
	var builder strings.Builder
	builder.Grow(length)

	max := big.NewInt(int64(len(slugAlphabet)))
	for i := 0; i < length; i++ {
		index, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}

		builder.WriteByte(slugAlphabet[index.Int64()])
	}

	return builder.String(), nil
}

func isDuplicateSlugError(err error) bool {
	if err == nil {
		return false
	}

	lowerErr := strings.ToLower(err.Error())
	return strings.Contains(lowerErr, "duplicate") ||
		strings.Contains(lowerErr, "unique") ||
		strings.Contains(lowerErr, "23505")
}

func (c *CliqService) CreateSlug(ctx context.Context, userID uuid.UUID, link dto.Link, shortLinkBase string) (dto.LinkResponse, error) {
	originLink := strings.TrimSpace(link.OriginLink)
	if err := validateOriginLink(originLink); err != nil {
		return dto.LinkResponse{}, err
	}

	customSlug := normalizeSlug(link.Slug)
	if customSlug != "" {
		if err := validateSlug(customSlug); err != nil {
			return dto.LinkResponse{}, err
		}

		created, err := c.repo.CreateSlug(ctx, userID, originLink, customSlug)
		if err != nil {
			if isDuplicateSlugError(err) {
				return dto.LinkResponse{}, ErrSlugAlreadyExists
			}

			return dto.LinkResponse{}, err
		}

		return toLinkResponse(created, shortLinkBase), nil
	}

	for i := 0; i < maxSlugRetries; i++ {
		slug, err := randomSlug(autoSlugLength)
		if err != nil {
			return dto.LinkResponse{}, err
		}

		created, err := c.repo.CreateSlug(ctx, userID, originLink, slug)
		if err == nil {
			return toLinkResponse(created, shortLinkBase), nil
		}

		if isDuplicateSlugError(err) {
			continue
		}

		return dto.LinkResponse{}, err
	}

	return dto.LinkResponse{}, ErrSlugAlreadyExists
}

func (c *CliqService) GetDashboard(ctx context.Context, userID uuid.UUID, page, limit int, shortLinkBase string) (dto.DashboardResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit
	links, total, err := c.repo.ListLinksByUser(ctx, userID, limit, offset)
	if err != nil {
		return dto.DashboardResponse{}, err
	}

	items := make([]dto.LinkResponse, 0, len(links))
	for _, link := range links {
		items = append(items, toLinkResponse(link, shortLinkBase))
	}

	totalPages := 1
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	return dto.DashboardResponse{
		Links:       items,
		TotalActive: total,
		TotalClicks: 0,
		Page:        page,
		Limit:       limit,
		TotalPages:  totalPages,
	}, nil
}

func (c *CliqService) DeleteLink(ctx context.Context, userID uuid.UUID, rawLinkID string) error {
	linkID, err := uuid.Parse(strings.TrimSpace(rawLinkID))
	if err != nil {
		return ErrInvalidLinkID
	}

	if err := c.repo.SoftDeleteLinkByID(ctx, userID, linkID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrLinkNotFound
		}

		return err
	}

	return nil
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

func toLinkResponse(link model.Link, shortLinkBase string) dto.LinkResponse {
	base := strings.TrimRight(shortLinkBase, "/")

	return dto.LinkResponse{
		ID:         link.ID.String(),
		OriginLink: link.OriginLink,
		Slug:       link.Slug,
		ShortURL:   base + "/" + link.Slug,
		Clicks:     0,
		CreatedAt:  link.CreatedAt,
	}
}
