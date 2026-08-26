// Package db 负责 GORM 初始化与自动迁移。
// 生产模式用 PostgreSQL, 本地模式用 SQLite, 由配置切换驱动。
package db

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

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
		&models.Admin{},
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
	// 旧版平台管理员(users.role=platform_admin)迁移到独立 admins 表。
	if err := migrateLegacyAdmins(gdb); err != nil {
		return err
	}
	return nil
}

// migrateLegacyAdmins 将旧版 users 表中的平台管理员迁移到 admins 表。
// 规则: users 表 role='platform_admin' 的行 → admins 表 role=super_admin, 迁移后从 users 删除。
// 若 admins 表已有数据(新版本已创建管理员), 只清理 users 中的残留行, 不覆盖新数据。
func migrateLegacyAdmins(gdb *gorm.DB) error {
	type legacyAdmin struct {
		Email        string
		PasswordHash string
		Nickname     string
		Status       string
		LastLoginAt  *time.Time
		CreatedAt    time.Time
		UpdatedAt    time.Time
	}
	var legacy []legacyAdmin
	if err := gdb.Model(&models.User{}).
		Where("role = ?", models.RolePlatformAdmin).
		Find(&legacy).Error; err != nil {
		return fmt.Errorf("迁移旧版平台管理员失败: %w", err)
	}
	if len(legacy) == 0 {
		return nil
	}

	// admins 表已有任何数据 → 只清理 users 残留, 避免覆盖新数据。
	var adminCount int64
	if err := gdb.Model(&models.Admin{}).Count(&adminCount).Error; err != nil {
		return fmt.Errorf("迁移旧版平台管理员失败: %w", err)
	}
	if adminCount > 0 {
		if err := gdb.Where("role = ?", models.RolePlatformAdmin).Delete(&models.User{}).Error; err != nil {
			return fmt.Errorf("迁移旧版平台管理员失败: %w", err)
		}
		return nil
	}

	// 事务内逐行复制(保留原时间戳), 然后删除 users 中的旧管理员。
	err := gdb.Transaction(func(tx *gorm.DB) error {
		for _, u := range legacy {
			nickname := u.Nickname
			if nickname == "" {
				nickname = "Administrator"
			}
			admin := models.Admin{
				Email:        u.Email,
				PasswordHash: u.PasswordHash,
				Nickname:     nickname,
				Role:         models.AdminRoleSuper,
				Status:       u.Status,
				LastLoginAt:  u.LastLoginAt,
				CreatedAt:    u.CreatedAt,
				UpdatedAt:    u.UpdatedAt,
			}
			if err := tx.Create(&admin).Error; err != nil {
				return err
			}
		}
		return tx.Where("role = ?", models.RolePlatformAdmin).Delete(&models.User{}).Error
	})
	if err != nil {
		return fmt.Errorf("迁移旧版平台管理员失败: %w", err)
	}
	return nil
}
