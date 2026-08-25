package channel

import (
	"context"
	"encoding/json"
	"fmt"

	"messagepusher/internal/models"
	"messagepusher/internal/queue"

	"gorm.io/gorm"
)

// SettingsReader 读取平台设置的接口(避免直接依赖 service 包造成循环)。
type SettingsReader interface {
	GetJSON(key string, target interface{}) error
}

// EmailSender 邮件渠道: SMTP 发送。
// 配置优先级: 租户自定义 Channel(type=email) > 平台 Settings(smtp)。
type EmailSender struct {
	db       *gorm.DB
	settings SettingsReader
}

// NewEmailSender 创建邮件渠道。
func NewEmailSender(db *gorm.DB, settings SettingsReader) *EmailSender {
	return &EmailSender{db: db, settings: settings}
}

// Type 渠道类型。
func (s *EmailSender) Type() string { return "email" }

// Send 发送邮件。
func (s *EmailSender) Send(ctx context.Context, msg *queue.TaskMessage) error {
	cfg, err := s.resolveConfig(ctx, msg.TenantID)
	if err != nil {
		return err
	}
	if msg.Recipient == "" {
		return fmt.Errorf("邮件收件人为空")
	}
	if msg.Title == "" && msg.Content == "" {
		return fmt.Errorf("邮件标题与内容为空")
	}
	return smtpSend(cfg, []string{msg.Recipient}, msg.Title, msg.Content)
}

// resolveConfig 解析 SMTP 配置: 租户覆盖优先, 否则平台默认。
func (s *EmailSender) resolveConfig(ctx context.Context, tenantID uint64) (SMTPConfig, error) {
	// 1. 租户自定义渠道配置。
	var ch models.Channel
	err := s.db.WithContext(ctx).Where("tenant_id = ? AND type = ? AND status = ?",
		tenantID, "email", models.StatusActive).First(&ch).Error
	if err == nil {
		var cfg SMTPConfig
		if err := json.Unmarshal([]byte(ch.Config), &cfg); err == nil && cfg.Host != "" {
			return cfg, nil
		}
	}

	// 2. 平台默认 SMTP(管理后台配置)。
	var cfg SMTPConfig
	if err := s.settings.GetJSON("smtp", &cfg); err != nil {
		return SMTPConfig{}, fmt.Errorf("读取平台 SMTP 配置失败: %w", err)
	}
	if cfg.Host == "" {
		return SMTPConfig{}, fmt.Errorf("邮件渠道未配置: 请先在管理后台配置 SMTP, 或在本租户渠道设置中配置")
	}
	return cfg, nil
}
