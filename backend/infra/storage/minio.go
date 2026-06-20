package storage

import (
	"context"
	"log/slog"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"vibe-studio/backend/conf"
)

// New 创建 MinIO(S3 兼容)客户端，并确保 bucket 存在。
func New(cfg conf.MinIOConfig) *minio.Client {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		slog.Warn("minio 初始化失败(degraded)", "err", err)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		slog.Warn("minio 连接失败(degraded)", "err", err)
		return client // 连接信息已建好，留待 /readyz 探测
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			slog.Warn("minio 创建 bucket 失败", "err", err)
		} else {
			slog.Info("minio 已创建 bucket", "bucket", cfg.Bucket)
		}
	}
	slog.Info("minio 连接成功")
	return client
}
