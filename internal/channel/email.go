package channel

import (
	"context"
	"encoding/json"
	"fmt"

	"axmipusher/internal/models"
	"axmipusher/internal/queue"

	"gorm.io/gorm"
)

// EmailSender 邮件渠道: SMTP 发送。
// 配置仅来自租户自定义 Channel(type=email), 不再降级平台 SMTP。
type EmailSender struct {
	db *gorm.DB
}

// NewEmailSender 创建邮件渠道。
func NewEmailSender(db *gorm.DB) *EmailSender {
	return &EmailSender{db: db}
}

// Type 渠道类型。
func (s *EmailSender) Type() string { return "email" }

// Send 发送邮件。
func (s *EmailSender) Send(ctx context.Context, msg *queue.TaskMessage) error {
	cfg, err := s.resolveConfig(ctx, msg.TenantID)
	if err != nil {
		return err
	}
	// 收件人: 发送时指定优先, 否则用通道配置的默认收件人。
	to := msg.Recipient
	if to == "" {
		to = cfg.Recipient
	}
	if to == "" {
		return fmt.Errorf("邮件收件人为空(发送时未指定, 且渠道未配置默认收件人)")
	}
	if msg.Title == "" && msg.Content == "" {
		return fmt.Errorf("邮件标题与内容为空")
	}
	return smtpSend(cfg, []string{to}, msg.Title, msg.Content)
}

// resolveConfig 解析 SMTP 配置: 仅租户自定义通道配置, 未配置即报错。
func (s *EmailSender) resolveConfig(ctx context.Context, tenantID uint64) (SMTPConfig, error) {
	var ch models.Channel
	err := s.db.WithContext(ctx).Where("tenant_id = ? AND type = ? AND status = ?",
		tenantID, "email", models.StatusActive).First(&ch).Error
	if err != nil {
		return SMTPConfig{}, fmt.Errorf("邮件渠道未配置: 请在用户中心-通道配置-邮件中设置 SMTP")
	}
	var cfg SMTPConfig
	if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
		return SMTPConfig{}, fmt.Errorf("解析邮件通道配置失败: %w", err)
	}
	if cfg.Host == "" {
		return SMTPConfig{}, fmt.Errorf("邮件渠道未配置: 请在用户中心-通道配置-邮件中设置 SMTP")
	}
	return cfg, nil
}
