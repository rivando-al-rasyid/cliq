package service

import (
	"github.com/redis/go-redis/v9"
)

type QliqRepository interface {
}

type QliqService struct {
	repo QliqRepository
	rdb  *redis.Client
}

func NewQliqService(repo QliqRepository, rdb *redis.Client) *QliqService {
	return &QliqService{repo: repo, rdb: rdb}
}
