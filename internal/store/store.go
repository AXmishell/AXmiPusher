// Package store 定义消息记录存储抽象。
// 消息存储与业务库同库(GORM), 消息写库即入队。
package store

import (
	"context"
	"time"
)

// Message 消息记录(状态机: PENDING→SENDING→SUCCESS/FAILED/RETRYING/DEAD)。
type Message struct {
	MessageID  uint64    `json:"message_id"`
	TenantID   uint64    `json:"tenant_id"`
	RequestID  string    `json:"request_id"`
	Channel    string    `json:"channel"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Recipient  string    `json:"recipient"`
	Status     string    `json:"status"`
	Error      string    `json:"error"`
	RetryCount int       `json:"retry_count"`
	CostMs     int64     `json:"cost_ms"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// MessageEvent 消息生命周期事件。
type MessageEvent struct {
	MessageID uint64    `json:"message_id"`
	EventType string    `json:"event_type"` // created|sending|success|failed|retry|dead
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

// MessageFilter 消息查询过滤条件。
type MessageFilter struct {
	Channel      string
	Status       string
	TemplateCode string
	Recipient    string
	Since        *time.Time
	Until        *time.Time
}

// MessageStore 消息记录存储接口。
type MessageStore interface {
	// SaveMessage 创建消息记录(返回分配的 MessageID)。
	SaveMessage(ctx context.Context, m *Message) (uint64, error)
	// UpdateStatus 更新消息状态与错误信息。
	UpdateStatus(ctx context.Context, messageID uint64, status, errMsg string) error
	// GetMessage 查询单条消息。
	GetMessage(ctx context.Context, messageID uint64) (*Message, error)
	// QueryMessages 分页查询(按创建时间倒序)。
	QueryMessages(ctx context.Context, tenantID uint64, f MessageFilter, page, size int) ([]*Message, int64, error)
	// SaveEvent 记录消息事件。
	SaveEvent(ctx context.Context, e *MessageEvent) error
	// StatsByStatus 按状态统计(时间范围)。
	StatsByStatus(ctx context.Context, tenantID uint64, since, until time.Time) (map[string]int64, error)
	// StatsByChannel 按渠道+状态统计(时间范围)。
	StatsByChannel(ctx context.Context, tenantID uint64, since, until time.Time) (map[string]map[string]int64, error)
	// ClaimPending 认领最多 limit 条 PENDING 消息(置为 SENDING 并刷新 updated_at 作租约), 返回被认领消息(含全量载荷)。
	ClaimPending(ctx context.Context, limit int) ([]*Message, error)
	// ReapStale 回收租约超时的 SENDING/RETRYING 消息: retry_count+1; 若 retry_count >= maxAttempts 置 DEAD, 否则复位 PENDING。返回处理条数。
	ReapStale(ctx context.Context, lease time.Duration, maxAttempts int) (int64, error)
	// Close 释放资源。
	Close() error
}
