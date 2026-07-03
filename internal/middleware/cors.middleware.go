package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

const defaultAllowedOrigins = "http://localhost,http://127.0.0.1,http://localhost:5173,http://127.0.0.1:5173,http://localhost:5500,http://127.0.0.1:5500"

func CORSMiddleware(ctx *gin.Context) {
	origin := ctx.GetHeader("Origin")

	allowedOrigins := getAllowedOrigins()

	if isOriginAllowed(origin, allowedOrigins) {
		ctx.Header("Access-Control-Allow-Origin", origin)
		ctx.Header("Vary", "Origin")
	}

	ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
	ctx.Header("Access-Control-Allow-Credentials", "true")

	if ctx.Request.Method == http.MethodOptions {
		ctx.AbortWithStatus(http.StatusNoContent)
		return
	}

	ctx.Next()
}

func getAllowedOrigins() map[string]bool {
	rawOrigins := os.Getenv("ALLOWED_ORIGINS")
	if strings.TrimSpace(rawOrigins) == "" {
		rawOrigins = os.Getenv("ALLOWED_ORIGIN")
	}

	if strings.TrimSpace(rawOrigins) == "" {
		rawOrigins = defaultAllowedOrigins
	}

	allowedOrigins := make(map[string]bool)

	for _, origin := range strings.Split(rawOrigins, ",") {
		normalizedOrigin := strings.TrimSpace(origin)

		if normalizedOrigin == "" {
			continue
		}

		allowedOrigins[normalizedOrigin] = true
	}

	return allowedOrigins
}

func isOriginAllowed(origin string, allowedOrigins map[string]bool) bool {
	if origin == "" {
		return false
	}

	return allowedOrigins[origin]
}
