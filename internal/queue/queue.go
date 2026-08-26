// Package queue 定义消息队列抽象。
// 队列基于业务库消息表轮询实现(消息写库即入队, 无独立队列中间件)。
package queue

import (
	"context"
	"time"
)

// TaskMessage 队列中传递的发送任务。
type TaskMessage struct {
	MessageID  uint64    `json:"message_id"`
	TenantID   uint64    `json:"tenant_id"`
	RequestID  string    `json:"request_id"`
	Channel    string    `json:"channel"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Recipient  string    `json:"recipient"`
	Priority   string    `json:"priority"`
	RetryCount int       `json:"retry_count"`
	CreatedAt  time.Time `json:"created_at"`
}

// Handler 消费处理函数, 返回 error 表示发送失败(触发重试)。
type Handler func(ctx context.Context, msg *TaskMessage) error

// Queue 消息队列接口。
type Queue interface {
	// Publish 发布一个发送任务。
	Publish(ctx context.Context, msg *TaskMessage) error
	// Subscribe 阻塞消费, 内部按并发数起 worker。
	Subscribe(ctx context.Context, handler Handler) error
	// Close 关闭队列释放资源。
	Close() error
	// Type 返回队列类型。
	Type() string
}
