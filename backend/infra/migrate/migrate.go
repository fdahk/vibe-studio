// Package migrate 用 golang-migrate + 内嵌 SQL 跑数据库迁移（替代 GORM AutoMigrate）。
// 版本化、可回滚、生产可控；迁移文件见 backend/migrations/。
package migrate

import (
	"database/sql"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	mysqldrv "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"vibe-studio/backend/migrations"
)

// Run 把数据库迁移到最新版本（已是最新则无操作）。
func Run(db *sql.DB) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}
	drv, err := mysqldrv.WithInstance(db, &mysqldrv.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", src, "mysql", drv)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
