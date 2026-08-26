// Package worker 消息消费者: 从队列取任务, 分发渠道发送, 更新状态。
package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"axmipusher/internal/channel"
	"axmipusher/internal/models"
	"axmipusher/internal/queue"
	"axmipusher/internal/store"
)

const (
	maxRetries   = 3
	retryBackoff = 2 * time.Second
)

// Worker 消费发送任务。
// 注意: store 与 registry 用惰性 getter —— 安装 Reinit 后会重建实例,
// 若构造时快照, 消费会写进旧 store(消息卡 PENDING) — 已踩坑。
type Worker struct {
	getStore    func() store.MessageStore
	getRegistry func() *channel.Registry
}

// New 创建 Worker。
func New(getStore func() store.MessageStore, getRegistry func() *channel.Registry) *Worker {
	return &Worker{getStore: getStore, getRegistry: getRegistry}
}

// Handle 处理单个发送任务(带同步重试)。
func (w *Worker) Handle(ctx context.Context, msg *queue.TaskMessage) {
	st := w.getStore()
	reg := w.getRegistry()
	start := time.Now()
	st.UpdateStatus(ctx, msg.MessageID, models.MsgSending, "")
	st.SaveEvent(ctx, &store.MessageEvent{MessageID: msg.MessageID, EventType: models.EventSending, CreatedAt: time.Now()})

	var err error
	attempt := 0
	for attempt <= maxRetries {
		if attempt > 0 {
			st.UpdateStatus(ctx, msg.MessageID, models.MsgRetrying, "")
			st.SaveEvent(ctx, &store.MessageEvent{
				MessageID: msg.MessageID, EventType: models.EventRetry,
				Detail: fmt.Sprintf("第 %d 次重试", attempt), CreatedAt: time.Now(),
			})
			select {
			case <-ctx.Done():
				return
			case <-time.After(retryBackoff * time.Duration(attempt)):
			}
		}
		err = reg.Dispatch(ctx, msg)
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
		st.UpdateStatus(ctx, msg.MessageID, models.MsgDead, err.Error())
		st.SaveEvent(ctx, &store.MessageEvent{
			MessageID: msg.MessageID, EventType: models.EventDead, Detail: err.Error(), CreatedAt: time.Now(),
		})
		return
	}
	st.UpdateStatus(ctx, msg.MessageID, models.MsgSuccess, "")
	st.SaveEvent(ctx, &store.MessageEvent{
		MessageID: msg.MessageID, EventType: models.EventSuccess,
		Detail: fmt.Sprintf("耗时 %dms", cost), CreatedAt: time.Now(),
	})
}
