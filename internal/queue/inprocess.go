package queue

import (
	"context"
	"fmt"
	"sync"
)

// InProcessQueue 进程内队列(本地模式)。
// 用带缓冲的 channel + 并发 worker 实现, 支持简单的重试入队。
type InProcessQueue struct {
	name        string
	bufferSize  int
	concurrency int
	ch          chan *TaskMessage
	wg          sync.WaitGroup
	mu          sync.Mutex
	closed      bool
	handler     Handler
}

// NewInProcessQueue 创建进程内队列。
func NewInProcessQueue(bufferSize, concurrency int) *InProcessQueue {
	if bufferSize <= 0 {
		bufferSize = 1024
	}
	if concurrency <= 0 {
		concurrency = 4
	}
	return &InProcessQueue{
		name:        "inprocess",
		bufferSize:  bufferSize,
		concurrency: concurrency,
		ch:          make(chan *TaskMessage, bufferSize),
	}
}

// Publish 入队。队列满时阻塞等待(生产者背压), ctx 取消则返回错误。
func (q *InProcessQueue) Publish(ctx context.Context, msg *TaskMessage) error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return fmt.Errorf("队列已关闭")
	}
	q.mu.Unlock()

	select {
	case q.ch <- msg:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("发布消息被取消: %w", ctx.Err())
	}
}

// Subscribe 启动消费者。
func (q *InProcessQueue) Subscribe(ctx context.Context, handler Handler) error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return fmt.Errorf("队列已关闭")
	}
	q.handler = handler
	q.mu.Unlock()

	for i := 0; i < q.concurrency; i++ {
		q.wg.Add(1)
		go q.worker(ctx, i)
	}
	<-ctx.Done()
	q.Close()
	return nil
}

func (q *InProcessQueue) worker(ctx context.Context, id int) {
	defer q.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-q.ch:
			if !ok {
				return
			}
			if q.handler == nil {
				continue
			}
			// 重试由上层 handler 自行负责(worker 内部带退避重试),
			// 队列层只负责投递, 防止重复消费造成重复发送。
			_ = q.handler(ctx, msg)
		}
	}
}

// Close 关闭队列。
func (q *InProcessQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return nil
	}
	q.closed = true
	close(q.ch)
	q.wg.Wait()
	return nil
}

// Type 返回队列类型。
func (q *InProcessQueue) Type() string { return q.name }
