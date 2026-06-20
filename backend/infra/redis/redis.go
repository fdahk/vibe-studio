package redis

import (
	"context"
	"log/slog"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"vibe-studio/backend/conf"
)

// New 创建 Redis 客户端。即使首次 ping 失败也返回 client（go-redis 会自动重连）。
func New(cfg conf.RedisConfig) *goredis.Client {
	client := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		slog.Warn("redis 连接失败(degraded)", "err", err)
	} else {
		slog.Info("redis 连接成功")
	}
	return client
}
