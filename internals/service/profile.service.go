package service

import (
	"context"
	"mime/multipart"
	"path"
	"strings"

	"github.com/rivando-al-rasyid/cliq/internals/config"
	"github.com/rivando-al-rasyid/cliq/internals/model"
	"github.com/rivando-al-rasyid/cliq/internals/pkg"
)

type ProfileRepository interface {
	UserProfile(ctx context.Context, email string) (model.Profile, error)
	GetUserInfo(ctx context.Context, email string) (model.Profile, int64, error)
	EditProfile(ctx context.Context, email string, updates map[string]any) (model.Profile, error)
	EditPassword(ctx context.Context, email string, newPassword string) (model.User, error)
	GetCurrentPassword(ctx context.Context, email string) (string, error)
}

type ProfileService struct {
	repo ProfileRepository
}

func NewProfileService(repo ProfileRepository) *ProfileService {
	return &ProfileService{repo: repo}
}

func (s *ProfileService) GetProfile(ctx context.Context, email string) (model.Profile, error) {
	return s.repo.UserProfile(ctx, email)
}

// GetUserInfo returns profile fields and total balance — used for the app header.
func (s *ProfileService) GetUserInfo(ctx context.Context, email string) (model.Profile, int64, error) {
	return s.repo.GetUserInfo(ctx, email)
}

func (s *ProfileService) EditProfile(ctx context.Context, email string, updates map[string]any) (model.Profile, error) {
	return s.repo.EditProfile(ctx, email, updates)
}

func (s *ProfileService) EditPassword(ctx context.Context, email, oldPassword, newPassword string) (model.User, error) {
	currentHash, err := s.repo.GetCurrentPassword(ctx, email)
	if err != nil {
		return model.User{}, err
	}
	var hc pkg.HashConfig
	if err := hc.Compare(oldPassword, currentHash); err != nil {
		return model.User{}, errWrongPassword
	}
	hc.UseRecommended()
	newHash := hc.GenHash(newPassword)
	return s.repo.EditPassword(ctx, email, newHash)
}

var errWrongPassword = errMsg("old password is incorrect")

type errMsg string

func (e errMsg) Error() string { return string(e) }

func (s *ProfileService) ValidateUpload(maxSize int64, photo *multipart.FileHeader) error {
	if photo.Size > maxSize {
		return config.ErrFileTooLarge
	}
	ext := strings.ToLower(path.Ext(photo.Filename))
	if !config.AllowedPhotoExt[ext] {
		return config.ErrExtNotAllowed
	}
	return nil
}
