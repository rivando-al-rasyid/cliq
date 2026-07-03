package controller

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rivando-al-rasyid/cliq-backend/internal/dto"
	"github.com/rivando-al-rasyid/cliq-backend/internal/pkg"
	"github.com/rivando-al-rasyid/cliq-backend/internal/service"
)

type LinkController struct {
	LinkService *service.LinkService
}

func NewLinkController(cliqService *service.LinkService) *LinkController {
	return &LinkController{LinkService: cliqService}
}

func shortLinkBase(ctx *gin.Context) string {
	if value := strings.TrimSpace(os.Getenv("SHORT_LINK_BASE_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}

	scheme := "http"
	if ctx.Request.TLS != nil || ctx.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	host := ctx.Request.Host
	if forwardedHost := strings.TrimSpace(ctx.GetHeader("X-Forwarded-Host")); forwardedHost != "" {
		host = forwardedHost
	}

	return scheme + "://" + host
}

// CreateSlug godoc
// @Summary      Create short link
// @Description  Creates a short link for the authenticated user. If the slug is empty, the service generates a random unique slug.
// @Tags         Links
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.Link      true  "Short link payload"
// @Success      201   {object}  dto.Response  "Short link created successfully"
// @Failure      400   {object}  dto.Response  "Invalid origin link, slug, or reserved slug"
// @Failure      401   {object}  dto.Response  "Unauthorized"
// @Failure      409   {object}  dto.Response  "Slug already exists"
// @Failure      500   {object}  dto.Response  "Internal server error"
// @Router       /link/create [post]
func (c *LinkController) CreateSlug(ctx *gin.Context) {
	userID, ok := pkg.CurrentUserID(ctx)
	if !ok {
		ctx.JSON(
			http.StatusUnauthorized,
			dto.NewError("Unauthorized", errors.New("missing or invalid user context")),
		)
		return
	}

	var body dto.Link
	if err := ctx.ShouldBindJSON(&body); err != nil {
		log.Printf("[LinkController.CreateSlug] bind error: %v\n", err)

		ctx.JSON(
			http.StatusBadRequest,
			dto.NewError("Invalid request payload", err),
		)
		return
	}

	link, err := c.LinkService.CreateSlug(ctx.Request.Context(), userID, body, shortLinkBase(ctx))
	if err != nil {
		log.Printf("[LinkController.CreateSlug] service error: %v\n", err)

		switch {
		case errors.Is(err, service.ErrInvalidOriginLink),
			errors.Is(err, service.ErrInvalidSlug),
			errors.Is(err, service.ErrReservedSlug):
			ctx.JSON(http.StatusBadRequest, dto.NewError("Invalid request payload", err))
		case errors.Is(err, service.ErrSlugAlreadyExists):
			ctx.JSON(http.StatusConflict, dto.NewError("Slug already exists", err))
		default:
			ctx.JSON(http.StatusInternalServerError, dto.NewError("Create slug failed", err))
		}
		return
	}

	ctx.JSON(
		http.StatusCreated,
		dto.NewSuccess("Short link created successfully", link),
	)
}

// GetDashboard godoc
// @Summary      Get link dashboard
// @Description  Returns dashboard totals and paginated active links for the authenticated user.
// @Tags         Links
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page   query     int  false  "Page number"     default(1)   minimum(1)
// @Param        limit  query     int  false  "Items per page"  default(10)  minimum(1)
// @Success      200    {object}  dto.Response{data=dto.DashboardResponse}  "Dashboard successfully retrieved"
// @Failure      401    {object}  dto.Response                            "Unauthorized"
// @Failure      500    {object}  dto.Response                            "Internal server error"
// @Router       /link/dashboard [get]
func (c *LinkController) GetDashboard(ctx *gin.Context) {
	userID, ok := pkg.CurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", errors.New("missing or invalid user context")))
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))

	dashboard, err := c.LinkService.GetDashboard(ctx.Request.Context(), userID, page, limit, shortLinkBase(ctx))
	if err != nil {
		log.Printf("[LinkController.GetDashboard] service error: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Dashboard failed", err))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccess("Dashboard successfully retrieved", dashboard))
}

// DeleteLink godoc
// @Summary      Delete short link
// @Description  Soft deletes a short link owned by the authenticated user.
// @Tags         Links
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string        true  "Link ID"
// @Success      200  {object}  dto.Response  "Link deleted successfully"
// @Failure      400  {object}  dto.Response  "Invalid link id"
// @Failure      401  {object}  dto.Response  "Unauthorized"
// @Failure      404  {object}  dto.Response  "Link not found"
// @Failure      500  {object}  dto.Response  "Internal server error"
// @Router       /link/{id} [delete]
func (c *LinkController) DeleteLink(ctx *gin.Context) {
	userID, ok := pkg.CurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", errors.New("missing or invalid user context")))
		return
	}

	if err := c.LinkService.DeleteLink(ctx.Request.Context(), userID, ctx.Param("id")); err != nil {
		log.Printf("[LinkController.DeleteLink] service error: %v\n", err)

		switch {
		case errors.Is(err, service.ErrInvalidLinkID):
			ctx.JSON(http.StatusBadRequest, dto.NewError("Invalid link id", err))
		case errors.Is(err, service.ErrLinkNotFound):
			ctx.JSON(http.StatusNotFound, dto.NewError("Link not found", err))
		default:
			ctx.JSON(http.StatusInternalServerError, dto.NewError("Delete link failed", err))
		}
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccessNoData("Link deleted successfully"))
}

// RedirectBySlug godoc
// @Summary      Redirect by slug
// @Description  Redirects to the original URL for the provided short link slug.
// @Tags         Links
// @Param        slug  path      string        true  "Short link slug"
// @Success      301   {string}  string        "Redirects to original URL"
// @Failure      404   {object}  dto.Response  "Slug not found"
// @Failure      500   {object}  dto.Response  "Internal server error"
// @Router       /{slug} [get]
func (c *LinkController) RedirectBySlug(ctx *gin.Context) {
	slug := ctx.Param("slug")

	originLink, err := c.LinkService.RedirectBySlug(ctx.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrLinkNotFound) {
			ctx.JSON(
				http.StatusNotFound,
				dto.NewError("Link not found", err),
			)
			return
		}

		log.Printf("[LinkController.RedirectBySlug] service error: %v\n", err)
		ctx.JSON(
			http.StatusInternalServerError,
			dto.NewError("Redirect failed", err),
		)
		return
	}

	ctx.Redirect(http.StatusMovedPermanently, originLink)
}
