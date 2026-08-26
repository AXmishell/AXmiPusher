package queue

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"messagepusher/internal/store"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestStore 创建 :memory: SQLite 测试存储(复用 store 包真实实现)。
// :memory: 单连接 —— MaxOpenConns 必须为 1, 否则多连接各自持有独立内存库。
func newTestStore(t *testing.T) *store.SQLiteStore {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开 :memory: SQLite 失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层连接池失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	s, err := store.NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteStore 失败: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return s
}

// newTestQueue 创建短间隔测试轮询队列(interval=20ms, 并发 2, 租约 1s)。
func newTestQueue(st store.MessageStore) *DBQueue {
	return NewDBQueue(func() store.MessageStore { return st }, 20*time.Millisecond, 10, 2, 5, time.Second)
}

// waitSubscribe 等待 Subscribe 退出(带超时防测试挂死)。
func waitSubscribe(t *testing.T, subErr <-chan error) {
	t.Helper()
	select {
	case <-subErr:
	case <-time.After(2 * time.Second):
		t.Fatalf("Subscribe 未在 2s 内退出, goroutine 泄漏")
	}
}

// ① 经 store.SaveMessage 插入 PENDING → 启动 Subscribe → handler 收到字段正确的
// TaskMessage, 且 DB 中消息被认领为 SENDING。
func TestDBQueueDeliversClaimedMessage(t *testing.T) {
	st := newTestStore(t)
	q := newTestQueue(st)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() { cancel(); q.Close() }()

	got := make(chan *TaskMessage, 1)
	subErr := make(chan error, 1)
	go func() {
		subErr <- q.Subscribe(ctx, func(c context.Context, tm *TaskMessage) error {
			select {
			case got <- tm:
			case <-c.Done():
			}
			return nil
		})
	}()

	now := time.Now()
	id, err := st.SaveMessage(context.Background(), &store.Message{
		TenantID: 7, RequestID: "req-7", Channel: "webhook",
		Title: "测试标题", Content: "测试内容", Recipient: "a@b.com",
		Status: "PENDING", CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("SaveMessage 失败: %v", err)
	}

	var tm *TaskMessage
	select {
	case tm = <-got:
	case <-time.After(2 * time.Second):
		t.Fatalf("2s 内未收到消息, 认领未触发")
	}
	// 断言字段映射正确。
	if tm.MessageID != id {
		t.Fatalf("MessageID 映射错误: 期望 %d, 实际 %d", id, tm.MessageID)
	}
	if tm.TenantID != 7 || tm.RequestID != "req-7" {
		t.Fatalf("TenantID/RequestID 映射错误: %+v", tm)
	}
	if tm.Channel != "webhook" || tm.Title != "测试标题" || tm.Content != "测试内容" || tm.Recipient != "a@b.com" {
		t.Fatalf("Channel/Title/Content/Recipient 映射错误: %+v", tm)
	}
	if tm.CreatedAt.Sub(now) > time.Second || tm.CreatedAt.Sub(now) < -time.Second {
		t.Fatalf("CreatedAt 映射错误: 期望约 %v, 实际 %v", now, tm.CreatedAt)
	}

	// 认领后 DB 中消息应变 SENDING。
	m, err := st.GetMessage(context.Background(), id)
	if err != nil {
		t.Fatalf("GetMessage 失败: %v", err)
	}
	if m.Status != "SENDING" {
		t.Fatalf("认领后应置 SENDING, 实际 %q", m.Status)
	}

	cancel()
	waitSubscribe(t, subErr)
}

// ② 取消 ctx 后 Subscribe 返回, 不泄漏 goroutine; 队列可重复关闭。
func TestDBQueueSubscribeReturnsOnCancel(t *testing.T) {
	st := newTestStore(t)
	q := newTestQueue(st)
	ctx, cancel := context.WithCancel(context.Background())
	subErr := make(chan error, 1)
	go func() {
		subErr <- q.Subscribe(ctx, func(c context.Context, tm *TaskMessage) error { return nil })
	}()

	cancel()
	waitSubscribe(t, subErr)
	// Close 幂等: 多次调用不 panic。
	q.Close()
	q.Close()
}

// ③ 空队列空转若干周期不 panic, 取消后正常退出。
func TestDBQueueEmptySpin(t *testing.T) {
	st := newTestStore(t)
	q := newTestQueue(st)
	ctx, cancel := context.WithCancel(context.Background())
	subErr := make(chan error, 1)
	go func() {
		subErr <- q.Subscribe(ctx, func(c context.Context, tm *TaskMessage) error { return nil })
	}()

	// 空队列空转 4 个周期。
	time.Sleep(80 * time.Millisecond)
	cancel()
	waitSubscribe(t, subErr)
}

// ④ 慢 handler(耗时 < 租约): 处理期间消息不被 ReapStale 回收(租约语义)。
func TestDBQueueLeaseProtectsInFlight(t *testing.T) {
	st := newTestStore(t)
	lease := 300 * time.Millisecond
	q := NewDBQueue(func() store.MessageStore { return st }, 20*time.Millisecond, 10, 2, 5, lease)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() { cancel(); q.Close() }()

	started := make(chan uint64, 1)
	release := make(chan struct{})
	subErr := make(chan error, 1)
	go func() {
		subErr <- q.Subscribe(ctx, func(c context.Context, tm *TaskMessage) error {
			started <- tm.MessageID
			<-release // 模拟慢 handler: 处理期间阻塞, 耗时 << lease。
			return nil
		})
	}()

	id, err := st.SaveMessage(context.Background(), &store.Message{
		TenantID: 1, RequestID: "r1", Channel: "webhook",
		Title: "t1", Content: "c1", Recipient: "a@b.com", Status: "PENDING",
	})
	if err != nil {
		t.Fatalf("SaveMessage 失败: %v", err)
	}

	// 等待 handler 进入处理(消息已被认领为 SENDING)。
	select {
	case gotID := <-started:
		if gotID != id {
			t.Fatalf("处理中的消息 ID 不一致: 期望 %d, 实际 %d", id, gotID)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("handler 未在 2s 内开始处理")
	}

	// 处理期间(租约未过期)ReapStale 不应回收。
	n, err := st.ReapStale(context.Background(), lease, 5)
	if err != nil {
		t.Fatalf("ReapStale 失败: %v", err)
	}
	if n != 0 {
		t.Fatalf("租约有效期内不应回收, 实际回收 %d 条", n)
	}
	m, err := st.GetMessage(context.Background(), id)
	if err != nil {
		t.Fatalf("GetMessage 失败: %v", err)
	}
	if m.Status != "SENDING" || m.RetryCount != 0 {
		t.Fatalf("处理中消息应保持 SENDING 且 retry_count=0, 实际 %q/%d", m.Status, m.RetryCount)
	}

	// 放行 handler 并等待 Subscribe 退出。
	close(release)
	cancel()
	waitSubscribe(t, subErr)
}

// ⑤ 并发批量: 插入 5 条, concurrency=2, 断言 5 条全部被处理且无重复。
func TestDBQueueConcurrentBatch(t *testing.T) {
	st := newTestStore(t)
	q := NewDBQueue(func() store.MessageStore { return st }, 20*time.Millisecond, 10, 2, 5, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() { cancel(); q.Close() }()

	var mu sync.Mutex
	var handled int32
	seen := make(map[uint64]int)
	subErr := make(chan error, 1)
	go func() {
		subErr <- q.Subscribe(ctx, func(c context.Context, tm *TaskMessage) error {
			atomic.AddInt32(&handled, 1)
			mu.Lock()
			seen[tm.MessageID]++
			mu.Unlock()
			return nil
		})
	}()

	const total = 5
	for i := 0; i < total; i++ {
		_, err := st.SaveMessage(context.Background(), &store.Message{
			TenantID: 1, RequestID: fmt.Sprintf("r%d", i), Channel: "webhook",
			Title: fmt.Sprintf("t%d", i), Content: fmt.Sprintf("c%d", i),
			Recipient: fmt.Sprintf("u%d@x.com", i), Status: "PENDING",
		})
		if err != nil {
			t.Fatalf("SaveMessage(%d) 失败: %v", i, err)
		}
	}

	// 等待 5 条全部处理(带超时防挂死)。
	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&handled) < total {
		if time.Now().After(deadline) {
			t.Fatalf("2s 内仅处理 %d/%d 条", atomic.LoadInt32(&handled), total)
		}
		time.Sleep(10 * time.Millisecond)
	}
	// 再空转若干周期, 确认无重复认领。
	time.Sleep(60 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if int(handled) != total {
		t.Fatalf("处理次数应为 %d, 实际 %d", total, handled)
	}
	if len(seen) != total {
		t.Fatalf("应处理 %d 条不同消息, 实际处理了 %d 条: %v", total, len(seen), seen)
	}
	for id, cnt := range seen {
		if cnt != 1 {
			t.Fatalf("消息 %d 被处理 %d 次, 应恰好 1 次(无重复)", id, cnt)
		}
	}
}
