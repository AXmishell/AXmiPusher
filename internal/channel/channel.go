// Package channel 定义渠道适配层。
// 每个渠道实现 Sender 接口, 由 Registry 统一分发。
package channel

import (
	"context"
	"fmt"

	"messagepusher/internal/queue"
)

// Sender 渠道发送器接口。
type Sender interface {
	// Type 返回渠道类型标识(email|apns|fcm|webhook|inapp)。
	Type() string
	// Send 发送消息, 返回 error 表示发送失败。
	Send(ctx context.Context, msg *queue.TaskMessage) error
}

// Registry 渠道注册表。
type Registry struct {
	senders map[string]Sender
	breaker Breaker
}

// NewRegistry 创建渠道注册表。
func NewRegistry() *Registry {
	return &Registry{senders: make(map[string]Sender)}
}

// SetBreaker 绑定熔断器(调用 Dispatch 前设置)。
func (r *Registry) SetBreaker(b Breaker) {
	r.breaker = b
}

// Register 注册渠道。
func (r *Registry) Register(s Sender) {
	r.senders[s.Type()] = s
}

// Dispatch 按消息渠道分发发送。渠道不存在或熔断时返回错误。
func (r *Registry) Dispatch(ctx context.Context, msg *queue.TaskMessage) error {
	s, ok := r.senders[msg.Channel]
	if !ok {
		return fmt.Errorf("渠道 %q 未注册", msg.Channel)
	}
	if r.breaker != nil {
		if !r.breaker.Allow(msg.TenantID, msg.Channel) {
			return &ErrCircuitOpen{Channel: msg.Channel}
		}
		err := s.Send(ctx, msg)
		if err != nil {
			r.breaker.Failure(msg.TenantID, msg.Channel)
		} else {
			r.breaker.Success(msg.TenantID, msg.Channel)
		}
		return err
	}
	return s.Send(ctx, msg)
}

// IsAvailable 渠道是否可用于 auto 路由: 已注册 且 未熔断。
func (r *Registry) IsAvailable(tenantID uint64, channel string) bool {
	if _, ok := r.senders[channel]; !ok {
		return false
	}
	if r.breaker != nil && r.breaker.IsOpen(tenantID, channel) {
		return false
	}
	return true
}

// HasChannel 判断渠道是否已注册。
func (r *Registry) HasChannel(typ string) bool {
	_, ok := r.senders[typ]
	return ok
}

// BreakerStats 渠道熔断观测数据(健康看板)。
func (r *Registry) BreakerStats(tenantID uint64, channel string) BreakerStats {
	if r.breaker != nil {
		return r.breaker.Stats(tenantID, channel)
	}
	return BreakerStats{State: "closed"}
}
