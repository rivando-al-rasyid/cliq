package config

import "errors"

const MB = 1 << 20

var AllowedPhotoExt = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
}

var (
	ErrFileTooLarge  = errors.New("file too large")
	ErrExtNotAllowed = errors.New("extension not allowed")
)
