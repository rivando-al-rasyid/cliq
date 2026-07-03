package pkg

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

const AccessTokenCookieName = "access_token"

func cookieSecure() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("COOKIE_SECURE")))
	if value == "true" || value == "1" || value == "yes" {
		return true
	}

	return strings.ToLower(os.Getenv("APP_ENV")) == "production" || strings.ToLower(os.Getenv("GIN_MODE")) == "release"
}

func cookieSameSite() http.SameSite {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("COOKIE_SAMESITE"))) {
	case "none":
		return http.SameSiteNoneMode
	case "strict":
		return http.SameSiteStrictMode
	default:
		return http.SameSiteLaxMode
	}
}

func SetAccessTokenCookie(ctx *gin.Context, token string) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     AccessTokenCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(AccessTokenExpiry.Seconds()),
		HttpOnly: true,
		Secure:   cookieSecure(),
		SameSite: cookieSameSite(),
	})
}

func ClearAccessTokenCookie(ctx *gin.Context) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     AccessTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cookieSecure(),
		SameSite: cookieSameSite(),
	})
}

func GetAccessTokenCookie(ctx *gin.Context) (string, error) {
	return ctx.Cookie(AccessTokenCookieName)
}
