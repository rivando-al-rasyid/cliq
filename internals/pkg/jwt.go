package pkg

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AccessTokenExpiry is the lifetime of an access JWT.
const AccessTokenExpiry = 24 * time.Hour

// ResetTokenExpiry is the lifetime of a short-lived password-reset JWT.
const ResetTokenExpiry = 10 * time.Minute

// ResetTokenSubject is the JWT "sub" claim used exclusively for password-reset JWTs.
// The change-password middleware checks for this value so a normal access token
// cannot be used to reach the change-password endpoint.
const ResetTokenSubject = "password-reset"

type Claims struct {
	ID    uuid.UUID
	Email string
	jwt.RegisteredClaims
}

func NewClaims(id uuid.UUID, email string) *Claims {
	return &Claims{
		ID:    id,
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    os.Getenv("JWT_ISSUER"),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenExpiry)),
		},
	}
}

// NewResetClaims returns a short-lived Claims scoped only for changing a password.
// The Subject field is set to ResetTokenSubject so the middleware can distinguish
// this token from a normal access token.
func NewResetClaims(id uuid.UUID, email string) *Claims {
	return &Claims{
		ID:    id,
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    os.Getenv("JWT_ISSUER"),
			Subject:   ResetTokenSubject,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ResetTokenExpiry)),
		},
	}
}

func (c *Claims) GenJWT() (string, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return "", errors.New("missing jwt secret")
	}
	uToken := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return uToken.SignedString([]byte(jwtSecret))
}

func (c *Claims) VerifyJWT(token string) error {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return errors.New("missing jwt secret")
	}
	log.Println("[JWT] Verifying token")
	jwtToken, err := jwt.ParseWithClaims(token, c, func(t *jwt.Token) (any, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return err
	}
	if !jwtToken.Valid {
		return jwt.ErrTokenExpired
	}
	iss, err := jwtToken.Claims.GetIssuer()
	if err != nil {
		return err
	}
	if iss != os.Getenv("JWT_ISSUER") {
		return jwt.ErrTokenInvalidIssuer
	}
	return nil
}
