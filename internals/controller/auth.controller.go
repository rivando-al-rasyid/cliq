package controller

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rivando-al-rasyid/vanwallet-backend/internals/dto"
	"github.com/rivando-al-rasyid/vanwallet-backend/internals/pkg"
	"github.com/rivando-al-rasyid/vanwallet-backend/internals/service"
)

type AuthController struct {
	authservice *service.AuthService
}

func NewAuthController(authservice *service.AuthService) *AuthController {
	return &AuthController{authservice: authservice}
}

// Register godoc
// @Summary      Register a new user
// @Description  Creates a new user account along with a profile, wallet, and PIN record.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        body  body      dto.RegisterRequest  true  "Registration credentials"
// @Success      201   {object}  dto.Response{data=dto.UserResponse}
// @Failure      400   {object}  dto.Response{error}
// @Failure      409   {object}  dto.Response{error}
// @Failure      500   {object}  dto.Response{error}
// @Router       /auth/register [post]
func (a *AuthController) Register(ctx *gin.Context) {
	var body dto.RegisterRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		log.Printf("[AuthController.Register] bind error: %v\n", err)
		ctx.JSON(http.StatusBadRequest, dto.NewError("Invalid request payload", "Please ensure your input matches the required format"))
		return
	}

	res, err := a.authservice.Register(ctx.Request.Context(), body)
	if err != nil {
		log.Printf("[AuthController.Register] service error: %v\n", err)
		status := http.StatusInternalServerError
		errDetail := "Internal server error"
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique") {
			status = http.StatusConflict
			errDetail = "Email already exists"
		}
		ctx.JSON(status, dto.NewError("Registration failed", errDetail))
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewSuccess("User successfully registered", res))
}

// Login godoc
// @Summary      Login
// @Description  Verifies email and password, then issues a signed JWT access token.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        body  body      dto.LoginRequest  true  "Login credentials"
// @Success      200   {object}  dto.Response{data=string}
// @Failure      400   {object}  dto.Response{error}
// @Failure      401   {object}  dto.Response{error}
// @Router       /auth/login [post]
func (a *AuthController) Login(ctx *gin.Context) {
	var body dto.LoginRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		log.Printf("[AuthController.Login] bind error: %v\n", err)
		ctx.JSON(http.StatusBadRequest, dto.NewError("Invalid request payload", "Please ensure your input matches the required format"))
		return
	}

	token, err := a.authservice.Login(ctx.Request.Context(), body)
	if err != nil {
		log.Printf("[AuthController.Login] service error: %v\n", err)
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Login failed", "Incorrect email or password"))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccess("Login successful", token))
}

// Logout godoc
// @Summary      Logout
// @Description  Revokes the current access token, invalidating the session.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.Response{data=object}
// @Failure      401  {object}  dto.Response{error}
// @Failure      500  {object}  dto.Response{error}
// @Router       /auth/logout [post]
func (a *AuthController) Logout(ctx *gin.Context) {
	claimsRaw, exists := ctx.Get("claims")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", "Missing claims"))
		return
	}
	email := claimsRaw.(pkg.Claims).Email

	rawToken, exists := ctx.Get("raw_token")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Logout failed", "raw token not found in context"))
		return
	}

	if err := a.authservice.Logout(ctx.Request.Context(), rawToken.(string), email); err != nil {
		log.Printf("[AuthController.Logout] service error: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Logout failed", "Internal server error"))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccessNoData("Logged out successfully"))
}

// GetPIN godoc
// @Summary      Get PIN status
// @Description  Returns whether a transaction PIN has been set for the authenticated user.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.Response{data=string}
// @Failure      401  {object}  dto.Response{error}
// @Failure      404  {object}  dto.Response{error}
// @Failure      500  {object}  dto.Response{error}
// @Router       /auth/pin [get]
func (a *AuthController) GetPIN(ctx *gin.Context) {
	claims, exists := ctx.Get("claims")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", "Missing claims"))
		return
	}

	email := claims.(pkg.Claims).Email
	pin, err := a.authservice.GetUserPin(ctx.Request.Context(), email)
	if err != nil {
		if err.Error() == "user pin not found" {
			ctx.JSON(http.StatusNotFound, dto.NewError("Failed to fetch PIN", "PIN not set for this user"))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Failed to fetch PIN", "Internal server error"))
		return
	}
	if pin.PinHash == nil || len(*pin.PinHash) == 0 {
		ctx.JSON(http.StatusNotFound, dto.NewError("Failed to fetch PIN", "PIN not set for this user"))
		return
	}
	ctx.JSON(http.StatusOK, dto.NewSuccess("PIN successfully retrieved", pin.PinHash))
}

// VerifyPIN godoc
// @Summary      Verify transaction PIN
// @Description  Validates the transaction PIN for the authenticated user.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.VerifyPinRequest  true  "PIN to verify"
// @Success      200   {object}  dto.Response{data=object}
// @Failure      400   {object}  dto.Response{error}
// @Failure      401   {object}  dto.Response{error}
// @Router       /auth/pin/verify [post]
func (a *AuthController) VerifyPIN(ctx *gin.Context) {
	claims, exists := ctx.Get("claims")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", "Missing claims"))
		return
	}
	email := claims.(pkg.Claims).Email

	var body dto.VerifyPinRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewError("Invalid request body", err.Error()))
		return
	}

	if err := a.authservice.VerifyPin(ctx.Request.Context(), email, body.Pin); err != nil {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Invalid PIN", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccessNoData("PIN verified successfully"))
}

// ResetPassword godoc
// @Summary      Request a password reset token
// @Description  Looks up the account by email and stores a short-lived PASSWORD_RESET token (5 min). Deliver this token to the user via email or SMS, then exchange it at POST /auth/reset/confirm.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        body  body      dto.ResetPasswordRequest  true  "Registered email address"
// @Success      201   {object}  dto.Response{data=string}
// @Failure      400   {object}  dto.Response{error}
// @Failure      404   {object}  dto.Response{error}
// @Failure      500   {object}  dto.Response{error}
// @Router       /auth/reset [post]
func (a *AuthController) ResetPassword(ctx *gin.Context) {
	var body dto.ResetPasswordRequest

	if err := ctx.ShouldBindBodyWithJSON(&body); err != nil {
		log.Printf("[AuthController.ResetPassword] bind error: %v\n", err)
		ctx.JSON(http.StatusBadRequest, dto.NewError("Invalid request payload", "Please ensure your input matches the required format"))
		return
	}
	token, err := a.authservice.ResetPassword(ctx.Request.Context(), body)
	if err != nil {
		log.Printf("[AuthController.ResetPassword] service error: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Reset password failed", "Internal server error"))
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewSuccess("Reset token generated", token))
}

// ConfirmResetPassword godoc
// @Summary      Confirm reset token and obtain a password-reset JWT
// @Description  Validates the opaque PASSWORD_RESET token issued by POST /auth/reset, revokes it (single-use), and returns a short-lived JWT (10 min, sub="password-reset"). Use this JWT as a Bearer token when calling POST /auth/change-password.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        body  body      dto.ConfirmResetPassword  true  "Reset token (from email/SMS)"
// @Success      200   {object}  dto.Response{data=string}
// @Failure      400   {object}  dto.Response{error}
// @Failure      401   {object}  dto.Response{error}
// @Failure      500   {object}  dto.Response{error}
// @Router       /auth/reset/confirm [post]
func (a *AuthController) ConfirmResetPassword(ctx *gin.Context) {
	var body dto.ConfirmResetPassword

	if err := ctx.ShouldBindBodyWithJSON(&body); err != nil {
		log.Printf("[AuthController.ConfirmResetPassword] bind error: %v\n", err)
		ctx.JSON(http.StatusBadRequest, dto.NewError("Invalid request payload", "Please ensure your input matches the required format"))
		return
	}

	resetJWT, err := a.authservice.ConfirmResetPassword(ctx.Request.Context(), body)
	if err != nil {
		log.Printf("[AuthController.ConfirmResetPassword] service error: %v\n", err)
		status := http.StatusInternalServerError
		errDetail := "Internal server error"
		if err.Error() == "invalid or expired reset token" {
			status = http.StatusUnauthorized
			errDetail = err.Error()
		}
		ctx.JSON(status, dto.NewError("Confirm reset failed", errDetail))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccess("Token confirmed. Use the returned JWT as Bearer token to call POST /auth/change-password", resetJWT))
}

// ChangePassword godoc
// @Summary      Set a new password using a password-reset JWT
// @Description  Replaces the user's password. Requires the short-lived JWT returned by POST /auth/reset/confirm as a Bearer token (sub must be "password-reset").
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.ChangeAndPasswordRequest  true  "New password"
// @Success      200   {object}  dto.Response{data=object}
// @Failure      400   {object}  dto.Response{error}
// @Failure      401   {object}  dto.Response{error}
// @Failure      500   {object}  dto.Response{error}
// @Router       /auth/change-password [post]
func (a *AuthController) ChangePassword(ctx *gin.Context) {
	claimsRaw, exists := ctx.Get("claims")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, dto.NewError("Unauthorized", "Missing claims"))
		return
	}
	claims := claimsRaw.(pkg.Claims)

	var body dto.ChangeAndPasswordRequest
	if err := ctx.ShouldBindBodyWithJSON(&body); err != nil {
		log.Printf("[AuthController.ChangePassword] bind error: %v\n", err)
		ctx.JSON(http.StatusBadRequest, dto.NewError("Invalid request payload", "Please ensure your input matches the required format"))
		return
	}

	if err := a.authservice.ChangeResetPassword(ctx.Request.Context(), claims.ID.String(), body.NewPassword); err != nil {
		log.Printf("[AuthController.ChangePassword] service error: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Change password failed", "Internal server error"))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccessNoData("Password changed successfully"))
}
