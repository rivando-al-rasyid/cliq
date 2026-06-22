package config

import (
	"context"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis() (*redis.Client, error) {
	redisAddr := os.Getenv("RDB_ADDR")

	var opt *redis.Options
	var err error

	if strings.HasPrefix(redisAddr, "redis://") || strings.HasPrefix(redisAddr, "rediss://") {
		opt, err = redis.ParseURL(redisAddr)
		if err != nil {
			return nil, err
		}
	} else {
		opt = &redis.Options{
			Addr:     redisAddr,
			Username: os.Getenv("RDB_USER"),
			Password: os.Getenv("RDB_PASS"),
		}
	}

	rc := redis.NewClient(opt)

	err = rc.Ping(context.Background()).Err()
	if err != nil {
		return nil, err
	}

	return rc, nil
}
