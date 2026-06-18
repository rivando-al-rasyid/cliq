package controller

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rivando-al-rasyid/cliq-backend/internals/dto"
	"github.com/rivando-al-rasyid/cliq-backend/internals/pkg"
	"github.com/rivando-al-rasyid/cliq-backend/internals/service"
)

type CliqController struct {
	CliqService *service.CliqService
}

func NewCliqController(cliqService *service.CliqService) *CliqController {
	return &CliqController{CliqService: cliqService}
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
// @Summary      Create a new short link
// @Description  Creates a shortened URL. If slug is empty, the system generates a random unique slug.
// @Tags         cliq
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body           body      dto.Link  true  "Slug Creation Payload"
// @Success      201            {object}  dto.Response "Short link created successfully"
// @Failure      400            {object}  dto.Response "Invalid request payload"
// @Failure      401            {object}  dto.Response "Unauthorized / Invalid token"
// @Failure      409            {object}  dto.Response "Slug already exists"
// @Failure      500            {object}  dto.Response "Internal server error"
// @Router       /link/create [post]
func (c *CliqController) CreateSlug(ctx *gin.Context) {
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
		log.Printf("[CliqController.CreateSlug] bind error: %v\n", err)

		ctx.JSON(
			http.StatusBadRequest,
			dto.NewError("Invalid request payload", err),
		)
		return
	}

	link, err := c.CliqService.CreateSlug(ctx.Request.Context(), userID, body, shortLinkBase(ctx))
	if err != nil {
		log.Printf("[CliqController.CreateSlug] service error: %v\n", err)

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
// @Summary      Get authenticated user's link dashboard
// @Description  Returns paginated active links and dashboard totals for the authenticated user.
// @Tags         cliq
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page   query     int  false  "Page number"
// @Param        limit  query     int  false  "Items per page"
// @Success      200    {object}  dto.Response{data=dto.DashboardResponse}
// @Failure      401    {object}  dto.Response "Unauthorized / Invalid token"
// @Failure      500    {object}  dto.Response "Internal server error"
// @Router       /link/dashboard [get]
func (c *CliqController) GetDashboard(ctx *gin.Context) {
	userID, ok := pkg.CurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", errors.New("missing or invalid user context")))
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))

	dashboard, err := c.CliqService.GetDashboard(ctx.Request.Context(), userID, page, limit, shortLinkBase(ctx))
	if err != nil {
		log.Printf("[CliqController.GetDashboard] service error: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Dashboard failed", err))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccess("Dashboard successfully retrieved", dashboard))
}

// DeleteLink godoc
// @Summary      Soft delete authenticated user's link
// @Description  Soft deletes a link owned by the authenticated user.
// @Tags         cliq
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Link ID"
// @Success      200  {object}  dto.Response "Link deleted successfully"
// @Failure      400  {object}  dto.Response "Invalid link id"
// @Failure      401  {object}  dto.Response "Unauthorized / Invalid token"
// @Failure      404  {object}  dto.Response "Link not found"
// @Failure      500  {object}  dto.Response "Internal server error"
// @Router       /link/{id} [delete]
func (c *CliqController) DeleteLink(ctx *gin.Context) {
	userID, ok := pkg.CurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", errors.New("missing or invalid user context")))
		return
	}

	if err := c.CliqService.DeleteLink(ctx.Request.Context(), userID, ctx.Param("id")); err != nil {
		log.Printf("[CliqController.DeleteLink] service error: %v\n", err)

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
// @Summary      Redirect short link
// @Description  Permanently redirects to the original URL based on the provided slug.
// @Tags         cliq
// @Produce      json
// @Param        slug  path      string  true  "Short link slug"
// @Success      301   {string}  string  "Redirects to original URL"
// @Failure      404   {object}  dto.Response "Slug not found"
// @Failure      500   {object}  dto.Response "Internal server error"
// @Router       /{slug} [get]
func (c *CliqController) RedirectBySlug(ctx *gin.Context) {
	slug := ctx.Param("slug")

	originLink, err := c.CliqService.RedirectBySlug(ctx.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrLinkNotFound) {
			ctx.JSON(
				http.StatusNotFound,
				dto.NewError("Link not found", err),
			)
			return
		}

		log.Printf("[CliqController.RedirectBySlug] service error: %v\n", err)
		ctx.JSON(
			http.StatusInternalServerError,
			dto.NewError("Redirect failed", err),
		)
		return
	}

	ctx.Redirect(http.StatusMovedPermanently, originLink)
}
