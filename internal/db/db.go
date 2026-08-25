// Package db 负责 GORM 初始化与自动迁移。
// 生产模式用 PostgreSQL, 本地模式用 SQLite, 由配置切换驱动。
package db

import (
	"fmt"
	"os"
	"path/filepath"

	"messagepusher/internal/config"
	"messagepusher/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open 按配置打开数据库连接并执行自动迁移。
func Open(cfg *config.Config) (*gorm.DB, error) {
	dsn, err := cfg.DSN()
	if err != nil {
		return nil, err
	}

	var dialector gorm.Dialector
	switch cfg.Database.Type {
	case "sqlite":
		if dir := filepath.Dir(dsn); dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("创建 SQLite 目录失败: %w", err)
			}
		}
		dialector = sqlite.Open(dsn)
	case "postgres":
		dialector = postgres.Open(dsn)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", cfg.Database.Type)
	}

	gdb, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	if err := migrate(gdb); err != nil {
		return nil, err
	}

	// SQLite 本地模式: 单写连接, 避免并发写锁冲突。
	if cfg.Database.Type == "sqlite" {
		if sqlDB, err := gdb.DB(); err == nil {
			sqlDB.SetMaxOpenConns(1)
		}
	}
	return gdb, nil
}

// migrate 自动迁移全部业务表。
func migrate(gdb *gorm.DB) error {
	if err := gdb.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.APIKey{},
		&models.CompatKey{},
		&models.Template{},
		&models.TemplateVersion{},
		&models.Channel{},
		&models.WebhookSubscription{},
		&models.BatchTask{},
		&models.Unsubscribe{},
		&models.AuditLog{},
		&models.Plan{},
		&models.Subscription{},
		&models.PaymentOrder{},
		&models.IdempotencyRecord{},
		&models.Setting{},
		&models.InappMessage{},
	); err != nil {
		return fmt.Errorf("自动迁移失败: %w", err)
	}
	return nil
}
