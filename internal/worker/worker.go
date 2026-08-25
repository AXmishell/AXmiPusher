// Package worker 消息消费者: 从队列取任务, 分发渠道发送, 更新状态。
package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"messagepusher/internal/channel"
	"messagepusher/internal/models"
	"messagepusher/internal/queue"
	"messagepusher/internal/store"
)

const (
	maxRetries   = 3
	retryBackoff = 2 * time.Second
)

// Worker 消费发送任务。
type Worker struct {
	store    store.MessageStore
	registry *channel.Registry
}

// New 创建 Worker。
func New(st store.MessageStore, registry *channel.Registry) *Worker {
	return &Worker{store: st, registry: registry}
}

// Handle 处理单个发送任务(带同步重试)。
func (w *Worker) Handle(ctx context.Context, msg *queue.TaskMessage) {
	start := time.Now()
	w.store.UpdateStatus(ctx, msg.MessageID, models.MsgSending, "")
	w.store.SaveEvent(ctx, &store.MessageEvent{MessageID: msg.MessageID, EventType: models.EventSending, CreatedAt: time.Now()})

	var err error
	attempt := 0
	for attempt <= maxRetries {
		if attempt > 0 {
			w.store.UpdateStatus(ctx, msg.MessageID, models.MsgRetrying, "")
			w.store.SaveEvent(ctx, &store.MessageEvent{
				MessageID: msg.MessageID, EventType: models.EventRetry,
				Detail: fmt.Sprintf("第 %d 次重试", attempt), CreatedAt: time.Now(),
			})
			select {
			case <-ctx.Done():
				return
			case <-time.After(retryBackoff * time.Duration(attempt)):
			}
		}
		err = w.registry.Dispatch(ctx, msg)
		if err == nil {
			break
		}
		// 熔断错误不重试(重试也会快速失败), 直接进入死信。
		var circuitErr *channel.ErrCircuitOpen
		if errors.As(err, &circuitErr) {
			break
		}
		attempt++
	}

	cost := time.Since(start).Milliseconds()
	if err != nil {
		w.store.UpdateStatus(ctx, msg.MessageID, models.MsgDead, err.Error())
		w.store.SaveEvent(ctx, &store.MessageEvent{
			MessageID: msg.MessageID, EventType: models.EventDead, Detail: err.Error(), CreatedAt: time.Now(),
		})
		return
	}
	w.store.UpdateStatus(ctx, msg.MessageID, models.MsgSuccess, "")
	w.store.SaveEvent(ctx, &store.MessageEvent{
		MessageID: msg.MessageID, EventType: models.EventSuccess,
		Detail: fmt.Sprintf("耗时 %dms", cost), CreatedAt: time.Now(),
	})
}
