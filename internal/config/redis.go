package config

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis() (*redis.Client, error) {
	redisAddr := strings.TrimSpace(os.Getenv("REDIS_URL"))
	if redisAddr == "" {
		redisAddr = strings.TrimSpace(os.Getenv("RDB_ADDR"))
	}

	if redisAddr == "" {
		return nil, errors.New("missing redis address: set REDIS_URL or RDB_ADDR")
	}

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

	if err := rc.Ping(context.Background()).Err(); err != nil {
		rc.Close()
		return nil, err
	}

	return rc, nil
}
