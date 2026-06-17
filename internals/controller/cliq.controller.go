package controller

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rivando-al-rasyid/cliq/internals/dto"
	"github.com/rivando-al-rasyid/cliq/internals/pkg"
	"github.com/rivando-al-rasyid/cliq/internals/service"
)

type CliqController struct {
	CliqService *service.CliqService
}

func NewCliqController(cliqService *service.CliqService) *CliqController {
	return &CliqController{CliqService: cliqService}
}

// CreateSlug godoc
// @Summary      Create a new short link
// @Description  Creates a shortened URL using the slug provided by the frontend.
// @Tags         cliq
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body           body      dto.Link  true  "Slug Creation Payload"
// @Success      201            {object}  dto.Response "Short link created successfully"
// @Failure      400            {object}  dto.Response "Invalid request payload"
// @Failure      401            {object}  dto.Response "Unauthorized / Invalid token"
// @Failure      500            {object}  dto.Response "Internal server error"
// @Router       /link/create [post]
func (c *CliqController) CreateSlug(ctx *gin.Context) {
	claimsRaw, exists := ctx.Get("claims")
	if !exists {
		ctx.JSON(
			http.StatusUnauthorized,
			dto.NewError("Unauthorized", errors.New("missing claims")),
		)
		return
	}

	claims, ok := claimsRaw.(*pkg.Claims)
	if !ok {
		ctx.JSON(
			http.StatusUnauthorized,
			dto.NewError("Unauthorized", errors.New("invalid claims")),
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

	slug, err := c.CliqService.CreateSlug(ctx.Request.Context(), claims.ID, body)
	if err != nil {
		log.Printf("[CliqController.CreateSlug] service error: %v\n", err)

		ctx.JSON(
			http.StatusInternalServerError,
			dto.NewError("Create slug failed", err),
		)
		return
	}

	ctx.JSON(
		http.StatusCreated,
		dto.NewSuccess("Short link created successfully", gin.H{
			"slug": slug,
		}),
	)
}

// RedirectBySlug godoc
// @Summary      Redirect short link
// @Description  Redirects to the original URL based on the provided slug.
// @Tags         cliq
// @Produce      json
// @Param        slug  path      string  true  "Short link slug"
// @Success      302   {string}  string  "Redirects to original URL"
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

	if !strings.HasPrefix(originLink, "http://") && !strings.HasPrefix(originLink, "https://") {
		originLink = "https://" + originLink
	}

	ctx.Redirect(http.StatusFound, originLink)
}
