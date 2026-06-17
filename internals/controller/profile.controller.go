package controller

import (
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/rivando-al-rasyid/cliq/internals/config"
	"github.com/rivando-al-rasyid/cliq/internals/dto"
	"github.com/rivando-al-rasyid/cliq/internals/pkg"
	"github.com/rivando-al-rasyid/cliq/internals/service"
)

type ProfileController struct {
	profileservice *service.ProfileService
}

func NewProfileController(profileservice *service.ProfileService) *ProfileController {
	return &ProfileController{profileservice: profileservice}
}

// GetProfile godoc
// @Summary      Get user profile details
// @Description  Retrieves current authentication details' full name, telephone connection code, and user avatar endpoint.
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200            {object}  dto.Response{data=dto.ProfileResponse}
// @Failure      401            {object}  dto.Response{error}
// @Failure      404            {object}  dto.Response{error}
// @Failure      500            {object}  dto.Response{error}
// @Router       /profile [get]
func (p *ProfileController) GetProfile(ctx *gin.Context) {
	claims, exists := ctx.Get("claims")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", errors.New("missing claims")))
		return
	}

	email := claims.(pkg.Claims).Email
	profile, err := p.profileservice.GetProfile(ctx.Request.Context(), email)
	if err != nil {
		if err.Error() == "user profile not found" {
			ctx.JSON(http.StatusNotFound, dto.NewError("Failed to fetch profile", errors.New("profile not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Failed to fetch profile", errors.New("internal server error")))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccess("Profile successfully retrieved", dto.ProfileResponse{
		FullName: profile.FullName,
		Phone:    profile.Phone,
		Photo:    profile.Photo,
	}))
}

// validateAndSavePhoto handles file validation and storage locally.
func (p *ProfileController) validateAndSavePhoto(ctx *gin.Context, photo *multipart.FileHeader, email string) (string, error) {
	if err := p.profileservice.ValidateUpload(2*config.MB, photo); err != nil {
		return "", err
	}

	ext := path.Ext(photo.Filename)
	filename := fmt.Sprintf("%s_photo_%d%s", strings.ToLower(strings.ReplaceAll(email, "@", "_")), time.Now().UnixNano(), ext)
	dst := filepath.Join("public", "img", filename)

	if err := ctx.SaveUploadedFile(photo, dst); err != nil {
		return "", err
	}

	return fmt.Sprintf("/img/%s", filename), nil
}

// EditProfile godoc
// @Summary      Modify active user profile records
// @Description  Updates system details including full name text, phone data info, and multipart image form attachments.
// @Tags         Profile
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        full_name      formData  string  false "Updated full identity name representation"
// @Param        phone          formData  string  false "Target telecommunications contact identity sequence"
// @Param        photo          formData  file    false "Binary source file image attachment content"
// @Success      200            {object}  dto.Response{data=object}
// @Failure      400            {object}  dto.Response{error}
// @Failure      401            {object}  dto.Response{error}
// @Failure      422            {object}  dto.Response{error}
// @Failure      500            {object}  dto.Response{error}
// @Router       /profile/edit [PATCH]
func (p *ProfileController) EditProfile(ctx *gin.Context) {
	claims, exists := ctx.Get("claims")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", errors.New("missing claims")))
		return
	}
	email := claims.(pkg.Claims).Email

	var body dto.UpdateProfileRequest
	if err := ctx.ShouldBindWith(&body, binding.FormMultipart); err != nil {
		log.Println("binding error: ", err.Error())
		ctx.JSON(http.StatusBadRequest, dto.NewError("Invalid request body", err))
		return
	}

	updates := map[string]any{}
	if body.FullName != nil {
		updates["full_name"] = body.FullName
	}
	if body.Phone != nil {
		updates["phone"] = body.Phone
	}
	if body.Photo != nil {
		photoURL, err := p.validateAndSavePhoto(ctx, body.Photo, email)
		if err != nil {
			log.Println("file handling error: ", err.Error())
			if errors.Is(err, config.ErrFileTooLarge) {
				ctx.JSON(http.StatusUnprocessableEntity, dto.NewError("File too large", errors.New("photo must be under 2MB")))
				return
			}
			if errors.Is(err, config.ErrExtNotAllowed) {
				ctx.JSON(http.StatusUnprocessableEntity, dto.NewError("Invalid file type", errors.New("only .jpg, .jpeg, .png, .webp are allowed")))
				return
			}
			ctx.JSON(http.StatusInternalServerError, dto.NewError("Error", errors.New("internal server error")))
			return
		}
		updates["photo"] = photoURL
	}

	_, err := p.profileservice.EditProfile(ctx.Request.Context(), email, updates)
	if err != nil {
		log.Println("service error: ", err.Error())
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Error", errors.New("internal server error")))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccessNoData("Profile successfully updated"))
}

// EditPassword godoc
// @Summary      Modify security entry password credentials
// @Description  Modifies internal account validation strings. Requires old confirmation verification strings.
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body           body      dto.ChangePasswordRequest  true  "Password structure swap payload"
// @Success      200            {object}  dto.Response{data=object}
// @Failure      400            {object}  dto.Response{error}
// @Failure      401            {object}  dto.Response{error}
// @Failure      500            {object}  dto.Response{error}
// @Router       /profile/password [PATCH]
func (p *ProfileController) EditPassword(ctx *gin.Context) {
	claims, exists := ctx.Get("claims")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", errors.New("missing claims")))
		return
	}
	email := claims.(pkg.Claims).Email

	var body dto.ChangePasswordRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewError("Invalid request body", err))
		return
	}

	_, err := p.profileservice.EditPassword(ctx.Request.Context(), email, body.OldPassword, body.Password)
	if err != nil {
		if err.Error() == "old password is incorrect" {
			ctx.JSON(http.StatusUnauthorized, dto.NewError("Failed to update password", errors.New("old password is incorrect")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Failed to update password", errors.New("internal server error")))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccessNoData("Password successfully updated"))
}

// GetUserInfo godoc
// @Summary      Get unified system user statistics context
// @Description  Assembles structured systemic components including identities, security states, and financial metrics.
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200            {object}  dto.Response{data=dto.UserInfoResponse}
// @Failure      401            {object}  dto.Response{error}
// @Failure      500            {object}  dto.Response{error}
// @Router       /profile/info [get]
func (p *ProfileController) GetUserInfo(ctx *gin.Context) {
	claims, exists := ctx.Get("claims")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", errors.New("missing claims")))
		return
	}
	claimsTyped := claims.(pkg.Claims)
	email := claimsTyped.Email

	profile, err := p.profileservice.GetUserInfo(ctx.Request.Context(), email)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Failed to fetch user info", errors.New("internal server error")))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccess("User info successfully retrieved", dto.UserInfoResponse{
		ID:       claimsTyped.ID.String(),
		Email:    email,
		FullName: profile.FullName,
		Phone:    profile.Phone,
		Photo:    profile.Photo,
	}))
}
