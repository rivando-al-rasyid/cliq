package controller

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rivando-al-rasyid/cliq-backend/internals/dto"
	"github.com/rivando-al-rasyid/cliq-backend/internals/pkg"
	"github.com/rivando-al-rasyid/cliq-backend/internals/service"
)

type AuthController struct {
	authservice *service.AuthService
}

func NewAuthController(authservice *service.AuthService) *AuthController {
	return &AuthController{authservice: authservice}
}

// Register godoc
// @Summary      Register a new user
// @Description  Creates a new user account and its default profile.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        body  body      dto.RegisterRequest  true  "Registration payload"
// @Success      201   {object}  dto.Response{data=dto.UserResponse}  "User successfully registered"
// @Failure      400   {object}  dto.Response                         "Invalid request payload"
// @Failure      409   {object}  dto.Response                         "Email already exists"
// @Failure      500   {object}  dto.Response                         "Internal server error"
// @Router       /auth/register [post]
func (a *AuthController) Register(ctx *gin.Context) {
	var body dto.RegisterRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		log.Printf("[AuthController.Register] bind error: %v\n", err)
		ctx.JSON(http.StatusBadRequest, dto.NewError("Invalid request payload", err))
		return
	}

	res, err := a.authservice.Register(ctx.Request.Context(), body)
	if err != nil {
		log.Printf("[AuthController.Register] service error: %v\n", err)
		status := http.StatusInternalServerError
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique") {
			status = http.StatusConflict
		}
		ctx.JSON(status, dto.NewError("Registration failed", err))
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewSuccess("User successfully registered", res))
}

// Login godoc
// @Summary      Login user
// @Description  Verifies email and password, stores the access JWT in an HttpOnly cookie, and returns public user data.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        body  body      dto.LoginRequest  true  "Login payload"
// @Success      200   {object}  dto.Response{data=dto.UserResponse}  "Login successful"
// @Header       200   {string}  Set-Cookie                           "HttpOnly access token cookie"
// @Failure      400   {object}  dto.Response                         "Invalid request payload"
// @Failure      401   {object}  dto.Response                         "Invalid email or password"
// @Router       /auth/login [post]
func (a *AuthController) Login(ctx *gin.Context) {
	var body dto.LoginRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		log.Printf("[AuthController.Login] bind error: %v\n", err)
		ctx.JSON(http.StatusBadRequest, dto.NewError("Invalid request payload", err))
		return
	}

	session, err := a.authservice.Login(ctx.Request.Context(), body)
	if err != nil {
		log.Printf("[AuthController.Login] service error: %v\n", err)
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Login failed", err))
		return
	}

	pkg.SetAccessTokenCookie(ctx, session.Token)

	ctx.JSON(http.StatusOK, dto.NewSuccess("Login successful", session.User))
}

// Me godoc
// @Summary      Get current authenticated user
// @Description  Returns the active user identity from the verified access cookie or Bearer token.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.Response{data=dto.UserResponse}  "Authenticated user retrieved"
// @Failure      401  {object}  dto.Response                         "Unauthorized"
// @Router       /auth/me [get]
func (a *AuthController) Me(ctx *gin.Context) {
	userID, ok := pkg.CurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", nil))
		return
	}

	email, ok := pkg.CurrentUserEmail(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", nil))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccess("Authenticated user retrieved", dto.UserResponse{
		ID:    userID,
		Email: email,
	}))
}

// Logout godoc
// @Summary      Logout user
// @Description  Revokes the current access token and clears the HttpOnly access cookie.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.Response  "Logged out successfully"
// @Header       200  {string}  Set-Cookie    "Cleared access token cookie"
// @Failure      401  {object}  dto.Response  "Unauthorized"
// @Failure      500  {object}  dto.Response  "Internal server error"
// @Router       /auth/logout [post]
func (a *AuthController) Logout(ctx *gin.Context) {
	defer pkg.ClearAccessTokenCookie(ctx)

	email, ok := pkg.CurrentUserEmail(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", nil))
		return
	}

	rawToken, ok := pkg.RawTokenFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Logout failed", nil))
		return
	}

	if err := a.authservice.Logout(ctx.Request.Context(), rawToken, email); err != nil {
		log.Printf("[AuthController.Logout] service error: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Logout failed", err))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccessNoData("Logged out successfully"))
}

// ResetPassword godoc
// @Summary      Request password reset token
// @Description  Looks up the account by email and stores a short-lived PASSWORD_RESET token. Deliver this token to the user via email or SMS, then exchange it at POST /auth/reset/confirm.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        body  body      dto.ResetPasswordRequest  true  "Registered email address"
// @Success      201   {object}  dto.Response              "Reset token generated"
// @Failure      400   {object}  dto.Response              "Invalid request payload"
// @Failure      404   {object}  dto.Response              "Account not found"
// @Failure      500   {object}  dto.Response              "Internal server error"
// @Router       /auth/reset [post]
func (a *AuthController) ResetPassword(ctx *gin.Context) {
	var body dto.ResetPasswordRequest

	if err := ctx.ShouldBindBodyWithJSON(&body); err != nil {
		log.Printf("[AuthController.ResetPassword] bind error: %v\n", err)
		ctx.JSON(http.StatusBadRequest, dto.NewError("Invalid request payload", err))
		return
	}
	token, err := a.authservice.ResetPassword(ctx.Request.Context(), body)
	if err != nil {
		log.Printf("[AuthController.ResetPassword] service error: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Reset password failed", err))
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewSuccess("Reset token generated", token))
}

// ConfirmResetPassword godoc
// @Summary      Confirm password reset token
// @Description  Validates the opaque PASSWORD_RESET token issued by POST /auth/reset, revokes it for single use, and returns a short-lived password-reset JWT.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        body  body      dto.ConfirmResetPassword  true  "Reset token payload"
// @Success      200   {object}  dto.Response              "Reset token confirmed"
// @Failure      400   {object}  dto.Response              "Invalid request payload"
// @Failure      401   {object}  dto.Response              "Invalid or expired reset token"
// @Failure      500   {object}  dto.Response              "Internal server error"
// @Router       /auth/reset/confirm [post]
func (a *AuthController) ConfirmResetPassword(ctx *gin.Context) {
	var body dto.ConfirmResetPassword

	if err := ctx.ShouldBindBodyWithJSON(&body); err != nil {
		log.Printf("[AuthController.ConfirmResetPassword] bind error: %v\n", err)
		ctx.JSON(http.StatusBadRequest, dto.NewError("Invalid request payload", err))
		return
	}

	resetJWT, err := a.authservice.ConfirmResetPassword(ctx.Request.Context(), body)
	if err != nil {
		log.Printf("[AuthController.ConfirmResetPassword] service error: %v\n", err)
		status := http.StatusInternalServerError
		if err.Error() == "invalid or expired reset token" {
			status = http.StatusUnauthorized
		}
		ctx.JSON(status, dto.NewError("Confirm reset failed", err))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccess("Token confirmed. Use the returned JWT as Bearer token to call POST /auth/change-password", resetJWT))
}

// ChangePassword godoc
// @Summary      Change password after reset confirmation
// @Description  Replaces the user's password. Requires the short-lived JWT returned by POST /auth/reset/confirm as a Bearer token.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.ChangeAndPasswordRequest  true  "New password payload"
// @Success      200   {object}  dto.Response                  "Password changed successfully"
// @Failure      400   {object}  dto.Response                  "Invalid request payload"
// @Failure      401   {object}  dto.Response                  "Unauthorized"
// @Failure      500   {object}  dto.Response                  "Internal server error"
// @Router       /auth/change-password [post]
func (a *AuthController) ChangePassword(ctx *gin.Context) {
	userID, ok := pkg.CurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", nil))
		return
	}

	var body dto.ChangeAndPasswordRequest
	if err := ctx.ShouldBindBodyWithJSON(&body); err != nil {
		log.Printf("[AuthController.ChangePassword] bind error: %v\n", err)
		ctx.JSON(http.StatusBadRequest, dto.NewError("Invalid request payload", err))
		return
	}

	if err := a.authservice.ChangeResetPassword(ctx.Request.Context(), userID, body.NewPassword); err != nil {
		log.Printf("[AuthController.ChangePassword] service error: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Change password failed", err))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccessNoData("Password changed successfully"))
}
