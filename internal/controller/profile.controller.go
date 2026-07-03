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
	"github.com/rivando-al-rasyid/cliq-backend/internals/config"
	"github.com/rivando-al-rasyid/cliq-backend/internals/dto"
	"github.com/rivando-al-rasyid/cliq-backend/internals/pkg"
	"github.com/rivando-al-rasyid/cliq-backend/internals/service"
)

type ProfileController struct {
	profileservice *service.ProfileService
}

func NewProfileController(profileservice *service.ProfileService) *ProfileController {
	return &ProfileController{profileservice: profileservice}
}

// GetProfile godoc
// @Summary      Get profile
// @Description  Retrieves the current user's profile, including full name, phone number, and photo URL.
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.Response{data=dto.ProfileResponse}  "Profile successfully retrieved"
// @Failure      401  {object}  dto.Response                            "Unauthorized"
// @Failure      404  {object}  dto.Response                            "Profile not found"
// @Failure      500  {object}  dto.Response                            "Internal server error"
// @Router       /profile [get]
func (p *ProfileController) GetProfile(ctx *gin.Context) {
	email, ok := pkg.CurrentUserEmail(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", errors.New("missing user context")))
		return
	}

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
// @Summary      Update profile
// @Description  Updates the current user's profile. Accepts optional full_name, phone, and photo multipart fields.
// @Tags         Profile
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        full_name  formData  string  false  "Full name"
// @Param        phone      formData  string  false  "Phone number"
// @Param        photo      formData  file    false  "Profile photo file (.jpg, .jpeg, .png, .webp, max 2MB)"
// @Success      200        {object}  dto.Response  "Profile successfully updated"
// @Failure      400        {object}  dto.Response  "Invalid request body"
// @Failure      401        {object}  dto.Response  "Unauthorized"
// @Failure      422        {object}  dto.Response  "Invalid file type or file too large"
// @Failure      500        {object}  dto.Response  "Internal server error"
// @Router       /profile/edit [patch]
func (p *ProfileController) EditProfile(ctx *gin.Context) {
	email, ok := pkg.CurrentUserEmail(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", errors.New("missing user context")))
		return
	}

	var body dto.UpdateProfileRequest
	if err := ctx.ShouldBindWith(&body, binding.FormMultipart); err != nil {
		log.Println("binding error: ", err.Error())
		ctx.JSON(http.StatusBadRequest, dto.NewError("Invalid request body", err))
		return
	}

	updates := map[string]any{}
	if body.FullName != nil {
		updates["full_name"] = *body.FullName
	}
	if body.Phone != nil {
		updates["phone"] = *body.Phone
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
// @Summary      Update profile password
// @Description  Changes the current user's password after validating the old password.
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.ChangePasswordRequest  true  "Password change payload"
// @Success      200   {object}  dto.Response               "Password successfully updated"
// @Failure      400   {object}  dto.Response               "Invalid request body"
// @Failure      401   {object}  dto.Response               "Unauthorized or old password is incorrect"
// @Failure      500   {object}  dto.Response               "Internal server error"
// @Router       /profile/password [patch]
func (p *ProfileController) EditPassword(ctx *gin.Context) {
	email, ok := pkg.CurrentUserEmail(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", errors.New("missing user context")))
		return
	}

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
// @Summary      Get user info
// @Description  Returns the authenticated user's account identity and profile summary.
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.Response{data=dto.UserInfoResponse}  "User info successfully retrieved"
// @Failure      401  {object}  dto.Response                             "Unauthorized"
// @Failure      500  {object}  dto.Response                             "Internal server error"
// @Router       /profile/info [get]
func (p *ProfileController) GetUserInfo(ctx *gin.Context) {
	email, ok := pkg.CurrentUserEmail(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", errors.New("missing user context")))
		return
	}
	userID, ok := pkg.CurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", errors.New("missing user context")))
		return
	}

	profile, err := p.profileservice.GetUserInfo(ctx.Request.Context(), email)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Failed to fetch user info", errors.New("internal server error")))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccess("User info successfully retrieved", dto.UserInfoResponse{
		ID:       userID.String(),
		Email:    email,
		FullName: profile.FullName,
		Phone:    profile.Phone,
		Photo:    profile.Photo,
	}))
}
