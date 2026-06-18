package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rivando-al-rasyid/cliq-backend/internals/cache"
	"github.com/rivando-al-rasyid/cliq-backend/internals/dto"
	"github.com/rivando-al-rasyid/cliq-backend/internals/model"
	"github.com/rivando-al-rasyid/cliq-backend/internals/pkg"
)

type AuthRepo interface {
	Register(ctx context.Context, email, password string) (model.User, error)
	Login(ctx context.Context, email string) (model.User, error)
	GetUserByResetToken(ctx context.Context, rawToken string) (model.User, error)
	SaveToken(ctx context.Context, userID uuid.UUID, rawToken string, tokenType model.TokenType, expiresAt time.Time) error
	RevokeToken(ctx context.Context, rawToken string) error
	IsTokenValid(ctx context.Context, rawToken string) (bool, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error
}

type AuthSession struct {
	Token string
	User  dto.UserResponse
}

type AuthService struct {
	authRepo AuthRepo
	rdb      *redis.Client
}

func NewAuthService(authRepo AuthRepo, rdb *redis.Client) *AuthService {
	return &AuthService{authRepo: authRepo, rdb: rdb}
}

func (a *AuthService) Register(ctx context.Context, user dto.RegisterRequest) (dto.UserResponse, error) {
	var hc pkg.HashConfig
	hc.UseRecommended()
	hashedPwd := hc.GenHash(user.Password)
	result, err := a.authRepo.Register(ctx, user.Email, hashedPwd)
	if err != nil {
		return dto.UserResponse{}, err
	}
	return dto.UserResponse{ID: result.ID, Email: result.Email}, nil
}

func (a *AuthService) Login(ctx context.Context, user dto.LoginRequest) (AuthSession, error) {
	login, err := a.getOrFetchUser(ctx, user.Email)
	if err != nil {
		return AuthSession{}, err
	}

	var hc pkg.HashConfig
	if err := hc.Compare(user.Password, login.Password); err != nil {
		return AuthSession{}, err
	}

	claims := pkg.NewClaims(login.ID, user.Email)
	token, err := claims.GenJWT()
	if err != nil {
		return AuthSession{}, err
	}
	expiresAt := time.Now().Add(pkg.AccessTokenExpiry)
	if err := a.authRepo.SaveToken(
		ctx,
		login.ID,
		token,
		model.TokenTypeAccess,
		expiresAt,
	); err != nil {
		return AuthSession{}, err
	}

	return AuthSession{
		Token: token,
		User: dto.UserResponse{
			ID:    login.ID,
			Email: user.Email,
		},
	}, nil
}

func (a *AuthService) ResetPassword(ctx context.Context, user dto.ResetPasswordRequest) (string, error) {
	login, err := a.getOrFetchUser(ctx, user.Email)
	if err != nil {
		return "", err
	}
	token, err := generateResetToken(32)
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(pkg.ResetTokenExpiry)
	if err := a.authRepo.SaveToken(
		ctx,
		login.ID,
		token,
		model.TokenTypePasswordReset,
		expiresAt,
	); err != nil {
		return "", err
	}

	return token, nil
}

func (a *AuthService) ConfirmResetPassword(ctx context.Context, user dto.ConfirmResetPassword) (string, error) {
	foundUser, err := a.authRepo.GetUserByResetToken(ctx, user.Token)
	if err != nil {
		return "", err
	}

	// Issue a short-lived JWT scoped exclusively for the change-password endpoint
	claims := pkg.NewResetClaims(foundUser.ID, foundUser.Email)
	resetJWT, err := claims.GenJWT()
	if err != nil {
		return "", err
	}

	return resetJWT, nil
}

// ChangeResetPassword hashes newPassword and persists it for the user identified by
// the password-reset JWT claims. The JWT was already validated (and the opaque
// reset token already revoked) in ConfirmResetPassword, so no extra token check
// is needed here.
func (a *AuthService) ChangeResetPassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	var hc pkg.HashConfig
	hc.UseRecommended()
	hashed := hc.GenHash(newPassword)
	return a.authRepo.UpdatePassword(ctx, userID, hashed)
}

func (a *AuthService) getOrFetchUser(ctx context.Context, email string) (*model.User, error) {
	rkey := "vando:user:" + email

	var user model.User
	if err := cache.GetFromCache(ctx, a.rdb, rkey, &user); err == nil {
		log.Println("cache hit:", email)
		return &user, nil
	} else if !errors.Is(err, redis.Nil) {
		log.Println("redis error:", err)
	}

	log.Println("cache miss:", email)
	fetched, err := a.authRepo.Login(ctx, email)
	if err != nil {
		return nil, err
	}

	if err := cache.SaveToCache(ctx, a.rdb, rkey, fetched); err != nil {
		log.Println("cache save error:", err) // non-fatal
	}

	return &fetched, nil
}

func (a *AuthService) Logout(ctx context.Context, rawToken, email string) error {
	if err := a.authRepo.RevokeToken(ctx, rawToken); err != nil {
		return err
	}
	rkey := "vando:user:" + email
	if err := cache.DelFromCache(ctx, a.rdb, rkey); err != nil {
		log.Println("cache evict error on logout:", err) // non-fatal
	}
	return nil
}

func generateResetToken(byteLength int) (string, error) {
	b := make([]byte, byteLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
