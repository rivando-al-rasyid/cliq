package config

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis() (*redis.Client, error) {
	rc := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("RDB_ADDR"),
		Username: os.Getenv("RDB_USER"),
		Password: os.Getenv("RDB_PASS"),
	})
	sc := rc.Ping(context.Background())
	return rc, sc.Err()
}
