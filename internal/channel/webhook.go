package channel

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"axmipusher/internal/models"
	"axmipusher/internal/queue"

	"gorm.io/gorm"
)

// WebhookSender Webhook 渠道: 将消息 POST 到租户注册的回调地址。
// 请求头带 HMAC-SHA256 签名, 业务方可验签防伪造。
type WebhookSender struct {
	db     *gorm.DB
	client *http.Client
}

// NewWebhookSender 创建 Webhook 渠道。
func NewWebhookSender(db *gorm.DB) *WebhookSender {
	return &WebhookSender{
		db: db,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Type 返回渠道类型。
func (s *WebhookSender) Type() string { return "webhook" }

// webhookPayload 发送到业务方回调地址的载荷。
type webhookPayload struct {
	MessageID uint64 `json:"message_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Recipient string `json:"recipient"`
	Timestamp int64  `json:"timestamp"`
}

// Send 发送 Webhook。
func (s *WebhookSender) Send(ctx context.Context, msg *queue.TaskMessage) error {
	var subs []models.WebhookSubscription
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ?", msg.TenantID, models.StatusActive).
		Find(&subs).Error; err != nil {
		return fmt.Errorf("查询回调订阅失败: %w", err)
	}
	if len(subs) == 0 {
		return fmt.Errorf("租户 %d 未配置 Webhook 回调地址", msg.TenantID)
	}

	payload := webhookPayload{
		MessageID: msg.MessageID,
		Title:     msg.Title,
		Content:   msg.Content,
		Recipient: msg.Recipient,
		Timestamp: time.Now().Unix(),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// 至少一个订阅成功才算成功。
	var lastErr error
	for _, sub := range subs {
		if err := s.post(ctx, sub, body); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("租户 %d 无可用回调地址", msg.TenantID)
}

// post 向单个订阅地址发送。
func (s *WebhookSender) post(ctx context.Context, sub models.WebhookSubscription, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "AXmiPusher/1.0")
	if sub.Secret != "" {
		mac := hmac.New(sha256.New, []byte(sub.Secret))
		mac.Write(body)
		req.Header.Set("X-MP-Signature", hex.EncodeToString(mac.Sum(nil)))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求 %s 失败: %w", sub.URL, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("回调 %s 返回非 2xx: %d", sub.URL, resp.StatusCode)
	}
	return nil
}
