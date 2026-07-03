package service

import (
	"context"
	"crypto/rand"
	"errors"
	"log"
	"math"
	"math/big"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rivando-al-rasyid/cliq-backend/internal/cache"
	"github.com/rivando-al-rasyid/cliq-backend/internal/dto"
	"github.com/rivando-al-rasyid/cliq-backend/internal/model"
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
	autoSlugLength     = 6
	maxSlugRetries     = 10
	slugAlphabet       = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	clickFlushInterval = 10 * time.Minute
	clickDirtySetKey   = "link:clicks:dirty"
)

var cleanupPendingClicksScript = redis.NewScript(`
local current = tonumber(redis.call("GET", KEYS[1]) or "0")
if current <= 0 then
	redis.call("DEL", KEYS[1])
	redis.call("SREM", KEYS[2], ARGV[1])
	return 1
end
return 0
`)

func linkRedirectCacheKey(slug string) string {
	return "link:redirect:" + slug
}

// linkClickCountKey stores pending clicks that have not been flushed to PostgreSQL yet.
func linkClickCountKey(slug string) string {
	return "link:clicks:" + slug
}

type LinkRepository interface {
	CreateSlug(ctx context.Context, userID uuid.UUID, originLink string, slug string) (model.Link, error)
	GetOriginLinkBySlug(ctx context.Context, slug string) (string, error)
	GetSlugByID(ctx context.Context, userID uuid.UUID, linkID uuid.UUID) (string, error)
	ListLinksByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Link, int, error)
	ListActiveSlugsByUser(ctx context.Context, userID uuid.UUID) ([]string, error)
	GetTotalClicksByUser(ctx context.Context, userID uuid.UUID) (int, error)
	IncrementClicksBySlug(ctx context.Context, slug string, delta int64) error
	SoftDeleteLinkByID(ctx context.Context, userID uuid.UUID, linkID uuid.UUID) (string, error)
}

type LinkService struct {
	repo LinkRepository
	rdb  *redis.Client
}

func NewLinkService(repo LinkRepository, rdb *redis.Client) *LinkService {
	return &LinkService{repo: repo, rdb: rdb}
}

func (c *LinkService) StartClickFlushWorker(ctx context.Context, interval time.Duration) {
	if c.rdb == nil {
		return
	}

	if interval <= 0 {
		interval = clickFlushInterval
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.FlushPendingClicks(context.Background()); err != nil {
					log.Printf("[LinkService.StartClickFlushWorker] flush pending clicks failed: %v\n", err)
				}
			}
		}
	}()
}

func (c *LinkService) cacheOriginLink(ctx context.Context, slug, originLink string) {
	if c.rdb == nil || slug == "" || originLink == "" {
		return
	}

	_ = cache.SaveToCache(ctx, c.rdb, linkRedirectCacheKey(slug), originLink)
}

func (c *LinkService) deleteLinkCaches(ctx context.Context, slug string) {
	if c.rdb == nil || slug == "" {
		return
	}

	_ = cache.DelFromCache(ctx, c.rdb, linkRedirectCacheKey(slug), linkClickCountKey(slug))
	_ = c.rdb.SRem(ctx, clickDirtySetKey, slug).Err()
}

func (c *LinkService) deleteRedirectCache(ctx context.Context, slug string) {
	if c.rdb == nil || slug == "" {
		return
	}

	_ = cache.DelFromCache(ctx, c.rdb, linkRedirectCacheKey(slug))
}

func (c *LinkService) resetClickCount(ctx context.Context, slug string) {
	if c.rdb == nil || slug == "" {
		return
	}

	_ = c.rdb.Del(ctx, linkClickCountKey(slug)).Err()
	_ = c.rdb.SRem(ctx, clickDirtySetKey, slug).Err()
}

func (c *LinkService) trackClick(ctx context.Context, slug string) {
	if c.rdb == nil || slug == "" {
		return
	}

	pipe := c.rdb.Pipeline()
	pipe.Incr(ctx, linkClickCountKey(slug))
	pipe.SAdd(ctx, clickDirtySetKey, slug)
	_, _ = pipe.Exec(ctx)
}

func parseRedisInt(value any) int {
	switch v := value.(type) {
	case nil:
		return 0
	case int:
		return v
	case int64:
		return int(v)
	case string:
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return 0
		}
		return parsed
	case []byte:
		parsed, err := strconv.Atoi(string(v))
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

func parseRedisInt64(value string) int64 {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return 0
	}

	return parsed
}

func (c *LinkService) getPendingClickCounts(ctx context.Context, slugs []string) map[string]int {
	counts := make(map[string]int, len(slugs))
	for _, slug := range slugs {
		counts[slug] = 0
	}

	if c.rdb == nil || len(slugs) == 0 {
		return counts
	}

	keys := make([]string, 0, len(slugs))
	for _, slug := range slugs {
		keys = append(keys, linkClickCountKey(slug))
	}

	values, err := c.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return counts
	}

	for i, value := range values {
		counts[slugs[i]] = parseRedisInt(value)
	}

	return counts
}

func sumPendingClicks(clickCounts map[string]int, slugs []string) int {
	total := 0
	for _, slug := range slugs {
		total += clickCounts[slug]
	}
	return total
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

func (c *LinkService) CreateSlug(ctx context.Context, userID uuid.UUID, link dto.Link, shortLinkBase string) (dto.LinkResponse, error) {
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

		c.cacheOriginLink(ctx, created.Slug, created.OriginLink)
		c.resetClickCount(ctx, created.Slug)

		return toLinkResponse(created, shortLinkBase, created.Clicks), nil
	}

	for i := 0; i < maxSlugRetries; i++ {
		slug, err := randomSlug(autoSlugLength)
		if err != nil {
			return dto.LinkResponse{}, err
		}

		created, err := c.repo.CreateSlug(ctx, userID, originLink, slug)
		if err == nil {
			c.cacheOriginLink(ctx, created.Slug, created.OriginLink)
			c.resetClickCount(ctx, created.Slug)

			return toLinkResponse(created, shortLinkBase, created.Clicks), nil
		}

		if isDuplicateSlugError(err) {
			continue
		}

		return dto.LinkResponse{}, err
	}

	return dto.LinkResponse{}, ErrSlugAlreadyExists
}

func (c *LinkService) GetDashboard(ctx context.Context, userID uuid.UUID, page, limit int, shortLinkBase string) (dto.DashboardResponse, error) {
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

	activeSlugs, err := c.repo.ListActiveSlugsByUser(ctx, userID)
	if err != nil {
		return dto.DashboardResponse{}, err
	}

	storedTotalClicks, err := c.repo.GetTotalClicksByUser(ctx, userID)
	if err != nil {
		return dto.DashboardResponse{}, err
	}

	pendingClickCounts := c.getPendingClickCounts(ctx, activeSlugs)
	items := make([]dto.LinkResponse, 0, len(links))
	for _, link := range links {
		clicks := link.Clicks + pendingClickCounts[link.Slug]
		items = append(items, toLinkResponse(link, shortLinkBase, clicks))
	}

	totalClicks := storedTotalClicks + sumPendingClicks(pendingClickCounts, activeSlugs)

	totalPages := 1
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	return dto.DashboardResponse{
		Links:       items,
		TotalActive: total,
		TotalClicks: totalClicks,
		Page:        page,
		Limit:       limit,
		TotalPages:  totalPages,
	}, nil
}

func (c *LinkService) DeleteLink(ctx context.Context, userID uuid.UUID, rawLinkID string) error {
	linkID, err := uuid.Parse(strings.TrimSpace(rawLinkID))
	if err != nil {
		return ErrInvalidLinkID
	}

	slug, err := c.repo.GetSlugByID(ctx, userID, linkID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrLinkNotFound
		}

		return err
	}

	// Stop serving old cached redirects before deleting the row.
	c.deleteRedirectCache(ctx, slug)

	// Persist pending Redis clicks before soft delete so they are not lost.
	if err := c.FlushPendingClicksBySlug(ctx, slug); err != nil {
		return err
	}

	if _, err := c.repo.SoftDeleteLinkByID(ctx, userID, linkID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrLinkNotFound
		}

		return err
	}

	// Capture any final clicks that arrived during the delete window.
	if err := c.FlushPendingClicksBySlug(ctx, slug); err != nil {
		return err
	}

	c.deleteLinkCaches(ctx, slug)

	return nil
}

func (c *LinkService) RedirectBySlug(ctx context.Context, slug string) (string, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return "", ErrLinkNotFound
	}

	if c.rdb != nil {
		var cachedOriginLink string
		err := cache.GetFromCache(ctx, c.rdb, linkRedirectCacheKey(slug), &cachedOriginLink)
		if err == nil && strings.TrimSpace(cachedOriginLink) != "" {
			c.trackClick(ctx, slug)
			return cachedOriginLink, nil
		}
	}

	originLink, err := c.repo.GetOriginLinkBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrLinkNotFound
		}

		return "", err
	}

	c.cacheOriginLink(ctx, slug, originLink)
	c.trackClick(ctx, slug)

	return originLink, nil
}

func (c *LinkService) FlushPendingClicks(ctx context.Context) error {
	if c.rdb == nil {
		return nil
	}

	slugs, err := c.rdb.SMembers(ctx, clickDirtySetKey).Result()
	if err != nil {
		return err
	}

	for _, slug := range slugs {
		if err := c.FlushPendingClicksBySlug(ctx, slug); err != nil {
			return err
		}
	}

	return nil
}

func (c *LinkService) FlushPendingClicksBySlug(ctx context.Context, slug string) error {
	if c.rdb == nil || strings.TrimSpace(slug) == "" {
		return nil
	}

	slug = strings.TrimSpace(slug)
	key := linkClickCountKey(slug)

	value, err := c.rdb.GetSet(ctx, key, "0").Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			_ = c.rdb.SRem(ctx, clickDirtySetKey, slug).Err()
			return nil
		}

		return err
	}

	delta := parseRedisInt64(value)
	if delta <= 0 {
		_, _ = cleanupPendingClicksScript.Run(ctx, c.rdb, []string{key, clickDirtySetKey}, slug).Result()
		return nil
	}

	if err := c.repo.IncrementClicksBySlug(ctx, slug, delta); err != nil {
		// Put the delta back so a temporary PostgreSQL failure does not lose clicks.
		pipe := c.rdb.Pipeline()
		pipe.IncrBy(ctx, key, delta)
		pipe.SAdd(ctx, clickDirtySetKey, slug)
		_, _ = pipe.Exec(ctx)

		return err
	}

	_, _ = cleanupPendingClicksScript.Run(ctx, c.rdb, []string{key, clickDirtySetKey}, slug).Result()

	return nil
}

func toLinkResponse(link model.Link, shortLinkBase string, clicks int) dto.LinkResponse {
	base := strings.TrimRight(shortLinkBase, "/")

	return dto.LinkResponse{
		ID:         link.ID.String(),
		OriginLink: link.OriginLink,
		Slug:       link.Slug,
		ShortURL:   base + "/" + link.Slug,
		Clicks:     clicks,
		CreatedAt:  link.CreatedAt,
	}
}
