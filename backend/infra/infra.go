package infra

import (
	"context"

	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"vibe-studio/backend/conf"
	mysqlinfra "vibe-studio/backend/infra/mysql"
	redisinfra "vibe-studio/backend/infra/redis"
	storageinfra "vibe-studio/backend/infra/storage"
)

// Deps 聚合所有基础设施客户端，统一向上层注入（手写依赖注入，右size：暂不引入 wire）。
type Deps struct {
	DB    *gorm.DB
	Redis *redis.Client
	MinIO *minio.Client
	Cfg   *conf.Config
}

// Init 初始化所有基础设施连接（任一失败不阻塞，见各子包的 degraded 说明）。
func Init(cfg *conf.Config) *Deps {
	return &Deps{
		DB:    mysqlinfra.New(cfg.MySQL),
		Redis: redisinfra.New(cfg.Redis),
		MinIO: storageinfra.New(cfg.MinIO),
		Cfg:   cfg,
	}
}

// HealthCheck 探测各依赖连通性，供 /readyz 使用。
func (d *Deps) HealthCheck(ctx context.Context) map[string]bool {
	checks := map[string]bool{"mysql": false, "redis": false, "minio": false}
	if d.DB != nil {
		if sqlDB, err := d.DB.DB(); err == nil && sqlDB.PingContext(ctx) == nil {
			checks["mysql"] = true
		}
	}
	if d.Redis != nil && d.Redis.Ping(ctx).Err() == nil {
		checks["redis"] = true
	}
	if d.MinIO != nil {
		if _, err := d.MinIO.ListBuckets(ctx); err == nil {
			checks["minio"] = true
		}
	}
	return checks
}
