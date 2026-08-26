package channel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"axmipusher/internal/models"
	"axmipusher/internal/queue"

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

	// 收件人解析(租户已折叠入用户, users 表无 tenant_id 列, 勿按 tenant_id 查):
	//   all → 发送者本人(该"租户"唯一用户); 否则按邮箱全局唯一查。
	var targets []models.User
	if msg.Recipient == "all" {
		var u models.User
		if err := s.db.WithContext(ctx).Where("id = ? AND status = ?",
			msg.TenantID, models.StatusActive).First(&u).Error; err != nil {
			return fmt.Errorf("查询发送者失败: %w", err)
		}
		targets = []models.User{u}
	} else {
		var u models.User
		err := s.db.WithContext(ctx).Where("email = ? AND status = ?",
			msg.Recipient, models.StatusActive).First(&u).Error
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
