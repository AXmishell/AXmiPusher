package queue

import (
	"context"
	"log"
	"sync"
	"time"

	"messagepusher/internal/store"
)

// DBQueue 数据库轮询队列。
// 消息不再经队列入队: service.EnqueueOne 已通过 store.SaveMessage 落库,
// 本队列按固定周期轮询数据库, 用 ClaimPending 认领 PENDING 消息交 worker 处理;
// 崩溃残留的 SENDING/RETRYING 消息由 ReapStale 按租约回收重试。
type DBQueue struct {
	// getStore 惰性取当前存储: Reinit 切库后仍拿到最新实例, 避免快照旧库导致消息静默丢失。
	getStore    func() store.MessageStore
	interval    time.Duration // 轮询间隔
	batch       int           // 每轮最大认领条数
	concurrency int           // 并发 worker 槽位数
	maxAttempts int           // 最大认领次数(超过置 DEAD)
	lease       time.Duration // 认领租约(消息处理超时后由 ReapStale 回收)
	done        chan struct{}
	wg          sync.WaitGroup
	closeOnce   sync.Once
}

// NewDBQueue 创建数据库轮询队列。
// getStore 必须传惰性函数而非 store 实例: 容器 Reinit 会重建存储,
// 若构造时快照, 轮询将一直读旧库, 切库后消息无人消费(已踩坑)。
func NewDBQueue(getStore func() store.MessageStore, interval time.Duration, batchSize, concurrency, maxAttempts int, lease time.Duration) *DBQueue {
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	if concurrency <= 0 {
		concurrency = 4
	}
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	if lease <= 0 {
		lease = 5 * time.Minute
	}
	return &DBQueue{
		getStore:    getStore,
		interval:    interval,
		batch:       batchSize,
		concurrency: concurrency,
		maxAttempts: maxAttempts,
		lease:       lease,
		done:        make(chan struct{}),
	}
}

// Publish 数据库轮询队列无需入队, 恒为 no-op。
// 消息已由 service.EnqueueOne → store.SaveMessage 落库为 PENDING,
// 轮询器按状态直接认领, 返回 nil 即视为受理成功。
func (q *DBQueue) Publish(ctx context.Context, msg *TaskMessage) error {
	return nil
}

// Subscribe 启动消费: 一个轮询 goroutine 按 interval 周期扫库,
// 认领到的消息交给 semaphore 槽位限流的并发 worker 处理。阻塞直到 ctx 取消或 Close。
func (q *DBQueue) Subscribe(ctx context.Context, handler Handler) error {
	sem := make(chan struct{}, q.concurrency)
	q.wg.Add(1)
	go q.poll(ctx, handler, sem)
	select {
	case <-ctx.Done():
	case <-q.done:
	}
	q.wg.Wait()
	return nil
}

// poll 轮询主循环: 每周期执行一轮认领, 直至 ctx 取消或队列关闭。
func (q *DBQueue) poll(ctx context.Context, handler Handler, sem chan struct{}) {
	defer q.wg.Done()
	ticker := time.NewTicker(q.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-q.done:
			return
		case <-ticker.C:
			q.round(ctx, handler, sem)
		}
	}
}

// round 单轮认领: 回收 → 按空闲槽位认领 → 认领即交 worker。
func (q *DBQueue) round(ctx context.Context, handler Handler, sem chan struct{}) {
	if ctx.Err() != nil {
		// 已取消: 正常关闭中, 不再执行本轮。
		return
	}
	st := q.getStore()
	if st == nil {
		// 存储尚未就绪(容器未构建), 本轮跳过。
		return
	}
	// ① 回收租约超时的 SENDING/RETRYING 消息; 失败仅记录, 不中断本轮。
	if _, err := st.ReapStale(ctx, q.lease, q.maxAttempts); err != nil {
		if ctx.Err() == nil {
			log.Printf("[queue:db] ReapStale 失败: %v", err)
		}
	}
	// ② 认领数量 = min(batch, 空闲槽位数); 无空闲槽位跳过本轮,
	// 认领即交 worker, 消除批量堆积。
	free := cap(sem) - len(sem)
	if free <= 0 {
		return
	}
	claim := q.batch
	if free < claim {
		claim = free
	}
	msgs, err := st.ClaimPending(ctx, claim)
	if err != nil {
		if ctx.Err() == nil {
			log.Printf("[queue:db] ClaimPending 失败: %v", err)
		}
		return
	}
	// ③ 认领到的每条消息交给空闲 worker 处理。
	for i := range msgs {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			return
		case <-q.done:
			return
		}
		m := msgs[i]
		q.wg.Add(1)
		go q.work(ctx, handler, sem, m)
	}
}

// work 执行单条消息的 handler, 占用一个并发槽位。
func (q *DBQueue) work(ctx context.Context, handler Handler, sem chan struct{}, m *store.Message) {
	defer q.wg.Done()
	defer func() { <-sem }()
	tm := &TaskMessage{
		MessageID:  m.MessageID,
		TenantID:   m.TenantID,
		RequestID:  m.RequestID,
		Channel:    m.Channel,
		Title:      m.Title,
		Content:    m.Content,
		Recipient:  m.Recipient,
		RetryCount: m.RetryCount,
		CreatedAt:  m.CreatedAt,
	}
	if err := handler(ctx, tm); err != nil {
		// 生产链路中 handler 内部(worker.Handle)已处理重试与终态(SUCCESS/DEAD)且不返回错误;
		// 此处错误分支为防御性保留, 消息若留 SENDING/RETRYING 由 ReapStale 回收。
		log.Printf("[queue:db] 处理消息 %d 失败: %v", m.MessageID, err)
	}
}

// Close 关闭队列: 通知轮询循环退出, 等待全部 worker 结束。幂等, 可多次调用。
func (q *DBQueue) Close() error {
	q.closeOnce.Do(func() { close(q.done) })
	q.wg.Wait()
	return nil
}

// Type 返回队列类型。
func (q *DBQueue) Type() string { return "db" }
