package mysql

import (
	"log/slog"

	driver "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"vibe-studio/backend/conf"
)

// New 尝试连接 MySQL。
// 设计选择(degraded 模式)：连不上时记录告警并返回 nil，而不是 panic 退出，
// 让 HTTP 服务即使没起 docker-compose 也能启动、/readyz 反映真实依赖状态。
func New(cfg conf.MySQLConfig) *gorm.DB {
	db, err := gorm.Open(driver.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		slog.Warn("mysql 连接失败(degraded)", "err", err)
		return nil
	}
	slog.Info("mysql 连接成功")
	return db
}
