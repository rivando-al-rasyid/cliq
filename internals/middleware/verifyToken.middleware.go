package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rivando-al-rasyid/cliq/internals/dto"
	"github.com/rivando-al-rasyid/cliq/internals/pkg"
	"github.com/rivando-al-rasyid/cliq/internals/repository"
)

func extractAndVerifyBearer(ctx *gin.Context, logTag string) (string, pkg.Claims, error) {
	bearerToken := ctx.GetHeader("Authorization")
	if bearerToken == "" {
		err := errors.New("missing authorization header")

		ctx.AbortWithStatusJSON(
			http.StatusUnauthorized,
			dto.NewError("Unauthorized", err),
		)

		return "", pkg.Claims{}, err
	}

	parts := strings.Fields(bearerToken)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		err := errors.New("invalid token format, use: Bearer <token>")

		ctx.AbortWithStatusJSON(
			http.StatusUnauthorized,
			dto.NewError("Unauthorized", err),
		)

		return "", pkg.Claims{}, err
	}

	rawToken := parts[1]

	var claims pkg.Claims
	if err := claims.VerifyJWT(rawToken); err != nil {
		log.Printf("[%s] JWT error: %v", logTag, err)

		if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenInvalidIssuer) {
			ctx.AbortWithStatusJSON(
				http.StatusUnauthorized,
				dto.NewError("Unauthorized", err),
			)

			return "", pkg.Claims{}, err
		}

		ctx.AbortWithStatusJSON(
			http.StatusUnauthorized,
			dto.NewError("Unauthorized", errors.New("invalid token")),
		)

		return "", pkg.Claims{}, err
	}

	return rawToken, claims, nil
}

// VerifyTokenWithDB validates the JWT signature and checks the tokens table
// token must exist, is_revoked = false, expires_at > now().
func VerifyTokenWithDB(db *pgxpool.Pool) gin.HandlerFunc {
	authRepo := repository.NewAuthRepo(db)

	return func(ctx *gin.Context) {
		rawToken, claims, err := extractAndVerifyBearer(ctx, "VerifyToken")
		if err != nil {
			return
		}

		valid, err := authRepo.IsTokenValid(context.Background(), rawToken)
		if err != nil {
			log.Println("[VerifyToken] DB token check error:", err)

			ctx.AbortWithStatusJSON(
				http.StatusInternalServerError,
				dto.NewError("Error", errors.New("internal server error")),
			)

			return
		}

		if !valid {
			ctx.AbortWithStatusJSON(
				http.StatusUnauthorized,
				dto.NewError(
					"Token has been revoked or expired, please login again",
					errors.New("token invalid"),
				),
			)

			return
		}

		ctx.Set("claims", &claims)
		ctx.Set("raw_token", rawToken)

		ctx.Next()
	}
}

// VerifyResetToken validates a JWT issued by ConfirmResetPassword.
// It enforces sub == pkg.ResetTokenSubject so that a normal access token
// cannot be used to reach the change-password endpoint.
// Reset JWTs are not stored in the tokens table, so no DB lookup is needed.
func VerifyResetToken() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		_, claims, err := extractAndVerifyBearer(ctx, "VerifyResetToken")
		if err != nil {
			return
		}

		if claims.Subject != pkg.ResetTokenSubject {
			ctx.AbortWithStatusJSON(
				http.StatusForbidden,
				dto.NewError(
					"Forbidden",
					errors.New("this token cannot be used for this operation"),
				),
			)

			return
		}

		ctx.Set("claims", &claims)

		ctx.Next()
	}
}
