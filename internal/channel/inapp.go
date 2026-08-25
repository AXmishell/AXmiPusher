package channel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"messagepusher/internal/models"
	"messagepusher/internal/queue"

	"gorm.io/gorm"
)

// InAppSender 站内信渠道: 将消息写入收件箱(租户内用户)。
// Recipient 为收件人邮箱; 特殊值 "all" 表示发送给租户全部用户。
type InAppSender struct {
	db *gorm.DB
}

// NewInAppSender 创建站内信渠道。
func NewInAppSender(db *gorm.DB) *InAppSender {
	return &InAppSender{db: db}
}

// Type 渠道类型。
func (s *InAppSender) Type() string { return "inapp" }

// Send 发送站内信。
func (s *InAppSender) Send(ctx context.Context, msg *queue.TaskMessage) error {
	if msg.Recipient == "" {
		return errors.New("站内信收件人为空")
	}
	now := time.Now()

	// 收件人解析: all → 全租户用户; 否则按邮箱查找用户。
	var targets []models.User
	if msg.Recipient == "all" {
		if err := s.db.WithContext(ctx).Where("tenant_id = ? AND status = ?",
			msg.TenantID, models.StatusActive).Find(&targets).Error; err != nil {
			return fmt.Errorf("查询租户用户失败: %w", err)
		}
	} else {
		var u models.User
		err := s.db.WithContext(ctx).Where("tenant_id = ? AND email = ? AND status = ?",
			msg.TenantID, msg.Recipient, models.StatusActive).First(&u).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("站内信收件人不存在: %s", msg.Recipient)
			}
			return err
		}
		targets = []models.User{u}
	}
	if len(targets) == 0 {
		return errors.New("站内信无有效收件人")
	}

	for _, u := range targets {
		rec := models.InappMessage{
			TenantID:  msg.TenantID,
			UserID:    u.ID,
			UserEmail: u.Email,
			Title:     msg.Title,
			Content:   msg.Content,
			IsRead:    false,
			CreatedAt: now,
		}
		if err := s.db.WithContext(ctx).Create(&rec).Error; err != nil {
			return fmt.Errorf("写入站内信失败: %w", err)
		}
	}
	return nil
}
