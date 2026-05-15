package db

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/mirdate/securegate/internal/config"
)

var rdb *redis.Client

func InitRedis(cfg *config.Config) error {
	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPass,
		DB:       0,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("Redis 연결 실패: %w", err)
	}

	return nil
}

func CloseRedis() {
	if rdb != nil {
		rdb.Close()
	}
}

func PingRedis() error {
	if rdb == nil {
		return fmt.Errorf("Redis 연결이 초기화되지 않았습니다")
	}
	return rdb.Ping(context.Background()).Err()
}

func Redis() *redis.Client {
	return rdb
}
