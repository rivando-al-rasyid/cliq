package pkg

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AccessTokenExpiry is the lifetime of an access JWT.
const AccessTokenExpiry = 15 * time.Minute

// ResetTokenExpiry is the lifetime of a short-lived password-reset JWT.
const ResetTokenExpiry = 10 * time.Minute

// AccessTokenSubject is the JWT "sub" claim used for normal access tokens.
const AccessTokenSubject = "access"

// ResetTokenSubject is the JWT "sub" claim used exclusively for password-reset JWTs.
const ResetTokenSubject = "password-reset"

const (
	ClaimsContextKey   = "claims"
	RawTokenContextKey = "raw_token"
)

type Claims struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	jwt.RegisteredClaims
}

func NewClaims(id uuid.UUID, email string) *Claims {
	return &Claims{
		ID:    id,
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    os.Getenv("JWT_ISSUER"),
			Subject:   AccessTokenSubject,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
}

func NewResetClaims(id uuid.UUID, email string) *Claims {
	return &Claims{
		ID:    id,
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    os.Getenv("JWT_ISSUER"),
			Subject:   ResetTokenSubject,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ResetTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
}

func (c *Claims) GenJWT() (string, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return "", errors.New("missing jwt secret")
	}

	jwtIssuer := os.Getenv("JWT_ISSUER")
	if jwtIssuer == "" {
		return "", errors.New("missing jwt issuer")
	}

	if c.Issuer == "" {
		c.Issuer = jwtIssuer
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)

	signedToken, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func (c *Claims) VerifyJWT(rawToken string) error {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return errors.New("missing jwt secret")
	}

	jwtIssuer := os.Getenv("JWT_ISSUER")
	if jwtIssuer == "" {
		return errors.New("missing jwt issuer")
	}

	token, err := jwt.ParseWithClaims(rawToken, c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}

		return []byte(jwtSecret), nil
	})
	if err != nil {
		return err
	}

	if !token.Valid {
		return jwt.ErrTokenInvalidClaims
	}

	if c.Issuer != jwtIssuer {
		return jwt.ErrTokenInvalidIssuer
	}

	return nil
}

func VerifyRawJWT(rawToken string) (Claims, error) {
	var claims Claims
	if err := claims.VerifyJWT(rawToken); err != nil {
		return Claims{}, err
	}

	return claims, nil
}

func ExtractBearerToken(authorizationHeader string) (string, error) {
	if strings.TrimSpace(authorizationHeader) == "" {
		return "", errors.New("missing authorization token")
	}

	parts := strings.Fields(authorizationHeader)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", errors.New("invalid token format, use: Bearer <token>")
	}

	return parts[1], nil
}

// ExtractRequestToken reads a token from Authorization: Bearer <token> first.
// If allowCookie is true, it falls back to the access_token HttpOnly cookie.
func ExtractRequestToken(ctx *gin.Context, allowCookie bool) (string, error) {
	if rawToken, err := ExtractBearerToken(ctx.GetHeader("Authorization")); err == nil {
		return rawToken, nil
	} else if strings.TrimSpace(ctx.GetHeader("Authorization")) != "" {
		return "", err
	}

	if allowCookie {
		cookieToken, err := GetAccessTokenCookie(ctx)
		if err == nil && strings.TrimSpace(cookieToken) != "" {
			return cookieToken, nil
		}
	}

	return "", errors.New("missing authorization token")
}

func SetAuthContext(ctx *gin.Context, rawToken string, claims Claims) {
	ctx.Set(ClaimsContextKey, claims)
	ctx.Set(RawTokenContextKey, rawToken)
}

func ClaimsFromContext(ctx *gin.Context) (Claims, bool) {
	claimsRaw, exists := ctx.Get(ClaimsContextKey)
	if !exists {
		return Claims{}, false
	}

	claims, ok := claimsRaw.(Claims)
	return claims, ok
}

func RawTokenFromContext(ctx *gin.Context) (string, bool) {
	rawToken, exists := ctx.Get(RawTokenContextKey)
	if !exists {
		return "", false
	}

	token, ok := rawToken.(string)
	return token, ok
}

func CurrentUserEmail(ctx *gin.Context) (string, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok || strings.TrimSpace(claims.Email) == "" {
		return "", false
	}

	return claims.Email, true
}

func CurrentUserID(ctx *gin.Context) (uuid.UUID, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok || claims.ID == uuid.Nil {
		return uuid.Nil, false
	}

	return claims.ID, true
}
