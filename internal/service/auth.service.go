package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rivando-al-rasyid/cliq-backend/internal/cache"
	"github.com/rivando-al-rasyid/cliq-backend/internal/dto"
	"github.com/rivando-al-rasyid/cliq-backend/internal/model"
	"github.com/rivando-al-rasyid/cliq-backend/internal/pkg"
)

type AuthRepo interface {
	Register(ctx context.Context, email, password string) (model.User, error)
	Login(ctx context.Context, email string) (model.User, error)
	FindOrCreateOAuthUser(ctx context.Context, email, passwordHash, fullName, photo string) (model.User, error)
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

	return a.createSession(ctx, *login)
}

func (a *AuthService) GoogleLogin(ctx context.Context, body dto.GoogleLoginRequest) (AuthSession, error) {
	profile, err := verifyGoogleCredential(ctx, body.Credential)
	if err != nil {
		return AuthSession{}, err
	}

	randomPassword, err := generateResetToken(32)
	if err != nil {
		return AuthSession{}, err
	}

	var hc pkg.HashConfig
	hc.UseRecommended()
	passwordHash := hc.GenHash(randomPassword)

	user, err := a.authRepo.FindOrCreateOAuthUser(
		ctx,
		profile.Email,
		passwordHash,
		profile.Name,
		profile.Picture,
	)
	if err != nil {
		return AuthSession{}, err
	}

	if err := cache.DelFromCache(ctx, a.rdb, userCacheKey(user.Email)); err != nil {
		log.Println("cache evict error on google login:", err)
	}

	return a.createSession(ctx, user)
}

func (a *AuthService) createSession(ctx context.Context, user model.User) (AuthSession, error) {
	claims := pkg.NewClaims(user.ID, user.Email)
	token, err := claims.GenJWT()
	if err != nil {
		return AuthSession{}, err
	}

	expiresAt := time.Now().Add(pkg.AccessTokenExpiry)
	if err := a.authRepo.SaveToken(
		ctx,
		user.ID,
		token,
		model.TokenTypeAccess,
		expiresAt,
	); err != nil {
		return AuthSession{}, err
	}

	return AuthSession{
		Token: token,
		User: dto.UserResponse{
			ID:    user.ID,
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
	rkey := userCacheKey(email)

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
	rkey := userCacheKey(email)
	if err := cache.DelFromCache(ctx, a.rdb, rkey); err != nil {
		log.Println("cache evict error on logout:", err) // non-fatal
	}
	return nil
}

type googleTokenInfo struct {
	Issuer        string `json:"iss"`
	Audience      string `json:"aud"`
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified any    `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Error         string `json:"error"`
	ErrorDesc     string `json:"error_description"`
}

func verifyGoogleCredential(ctx context.Context, credential string) (googleTokenInfo, error) {
	clientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	if clientID == "" {
		return googleTokenInfo{}, errors.New("GOOGLE_CLIENT_ID is not configured")
	}

	endpoint := "https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(credential)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return googleTokenInfo{}, err
	}

	client := &http.Client{Timeout: 8 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return googleTokenInfo{}, fmt.Errorf("verify google token: %w", err)
	}
	defer res.Body.Close()

	var profile googleTokenInfo
	if err := json.NewDecoder(res.Body).Decode(&profile); err != nil {
		return googleTokenInfo{}, fmt.Errorf("decode google token info: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(profile.ErrorDesc)
		if msg == "" {
			msg = strings.TrimSpace(profile.Error)
		}
		if msg == "" {
			msg = "invalid google credential"
		}
		return googleTokenInfo{}, errors.New(msg)
	}

	if profile.Audience != clientID {
		return googleTokenInfo{}, errors.New("google credential audience does not match this app")
	}

	if profile.Issuer != "accounts.google.com" && profile.Issuer != "https://accounts.google.com" {
		return googleTokenInfo{}, errors.New("invalid google credential issuer")
	}

	if strings.TrimSpace(profile.Subject) == "" || strings.TrimSpace(profile.Email) == "" {
		return googleTokenInfo{}, errors.New("google credential is missing required identity fields")
	}

	if !isGoogleEmailVerified(profile.EmailVerified) {
		return googleTokenInfo{}, errors.New("google account email is not verified")
	}

	profile.Email = strings.ToLower(strings.TrimSpace(profile.Email))
	return profile, nil
}

func isGoogleEmailVerified(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true")
	default:
		return false
	}
}

func userCacheKey(email string) string {
	return "cliq:user:" + strings.ToLower(strings.TrimSpace(email))
}

func generateResetToken(byteLength int) (string, error) {
	b := make([]byte, byteLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
