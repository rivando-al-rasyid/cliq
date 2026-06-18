package middleware

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rivando-al-rasyid/cliq/internals/dto"
	"github.com/rivando-al-rasyid/cliq/internals/pkg"
	"github.com/rivando-al-rasyid/cliq/internals/repository"
)

func extractAndVerifyToken(ctx *gin.Context, logTag string, allowCookie bool) (string, pkg.Claims, error) {
	rawToken, err := pkg.ExtractRequestToken(ctx, allowCookie)
	if err != nil {
		ctx.AbortWithStatusJSON(
			http.StatusUnauthorized,
			dto.NewError("Unauthorized", err),
		)

		return "", pkg.Claims{}, err
	}

	claims, err := pkg.VerifyRawJWT(rawToken)
	if err != nil {
		log.Printf("[%s] JWT error: %v", logTag, err)

		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			ctx.AbortWithStatusJSON(
				http.StatusUnauthorized,
				dto.NewError("Token expired", err),
			)

		case errors.Is(err, jwt.ErrTokenInvalidIssuer):
			ctx.AbortWithStatusJSON(
				http.StatusUnauthorized,
				dto.NewError("Invalid token issuer", err),
			)

		default:
			ctx.AbortWithStatusJSON(
				http.StatusUnauthorized,
				dto.NewError("Invalid token", errors.New("invalid token")),
			)
		}

		return "", pkg.Claims{}, err
	}

	return rawToken, claims, nil
}

// AuthRequired validates normal access JWT and checks the tokens table.
// The token can come from either Authorization: Bearer <token> or the
// HttpOnly access_token cookie used by the React Router data-mode frontend.
func AuthRequired(db *pgxpool.Pool) gin.HandlerFunc {
	authRepo := repository.NewAuthRepo(db)

	return func(ctx *gin.Context) {
		rawToken, claims, err := extractAndVerifyToken(ctx, "AuthRequired", true)
		if err != nil {
			return
		}

		if claims.Subject != pkg.AccessTokenSubject {
			ctx.AbortWithStatusJSON(
				http.StatusForbidden,
				dto.NewError(
					"Forbidden",
					errors.New("this token cannot be used for normal access"),
				),
			)

			return
		}

		valid, err := authRepo.IsTokenValid(ctx.Request.Context(), rawToken)
		if err != nil {
			log.Println("[AuthRequired] DB token check error:", err)

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

		pkg.SetAuthContext(ctx, rawToken, claims)

		ctx.Next()
	}
}

// PasswordResetRequired validates a JWT issued for reset password.
// Reset JWTs should be sent explicitly through Authorization: Bearer <token>,
// not the access_token cookie.
func PasswordResetRequired() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		_, claims, err := extractAndVerifyToken(ctx, "PasswordResetRequired", false)
		if err != nil {
			return
		}

		if claims.Subject != pkg.ResetTokenSubject {
			ctx.AbortWithStatusJSON(
				http.StatusForbidden,
				dto.NewError(
					"Forbidden",
					errors.New("this token cannot be used for password reset"),
				),
			)

			return
		}

		pkg.SetAuthContext(ctx, "", claims)

		ctx.Next()
	}
}

// VerifyTokenWithDB is kept for compatibility. Prefer AuthRequired in routers.
func VerifyTokenWithDB(db *pgxpool.Pool) gin.HandlerFunc {
	return AuthRequired(db)
}

// VerifyResetToken is kept for compatibility. Prefer PasswordResetRequired in routers.
func VerifyResetToken() gin.HandlerFunc {
	return PasswordResetRequired()
}
