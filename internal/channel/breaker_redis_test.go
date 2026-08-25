package channel

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// newTestRedis 创建 miniredis 客户端。
func newTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })
	return mr, rdb
}

// TestRedisCircuitBreaker Redis 熔断状态机与内存版一致。
// 注意: 脚本用 Go 墙钟做冷却判断, 因此用真实小冷却 + sleep, 不用 miniredis.FastForward。
func TestRedisCircuitBreaker(t *testing.T) {
	_, rdb := newTestRedis(t)
	b := NewRedisCircuitBreaker(rdb, 3, 20*time.Millisecond)
	const tid = uint64(1)

	// 闭合: 放行, 失败 3 次熔断。
	if !b.Allow(tid, "email") {
		t.Fatal("初始应放行")
	}
	b.Failure(tid, "email")
	b.Failure(tid, "email")
	if b.IsOpen(tid, "email") {
		t.Fatal("2 次失败不应熔断")
	}
	if !b.Allow(tid, "email") {
		t.Fatal("未熔断时应放行")
	}
	b.Failure(tid, "email")
	if !b.IsOpen(tid, "email") {
		t.Fatal("3 次失败后应熔断")
	}
	if b.Allow(tid, "email") {
		t.Fatal("熔断期内应拒绝")
	}

	// 冷却结束 → 半开放行探针。
	time.Sleep(40 * time.Millisecond)
	if !b.Allow(tid, "email") {
		t.Fatal("冷却后应放行探针")
	}
	b.Success(tid, "email")
	if b.IsOpen(tid, "email") {
		t.Fatal("探针成功后应闭合")
	}

	// 探针失败 → 立即重新熔断。
	b.Failure(tid, "email")
	b.Failure(tid, "email")
	b.Failure(tid, "email")
	time.Sleep(40 * time.Millisecond)
	b.Allow(tid, "email")
	b.Failure(tid, "email")
	if !b.IsOpen(tid, "email") {
		t.Fatal("半开探针失败后应立即重新熔断")
	}

	// 租户隔离。
	if b.IsOpen(2, "email") {
		t.Fatal("熔断应按租户隔离")
	}

	// Stats 观测。
	st := b.Stats(tid, "email")
	if st.State != "open" {
		t.Fatalf("状态应为 open, 实际 %s", st.State)
	}
	if st.TotalFailures < 4 {
		t.Fatalf("total_failures 应 >=4, 实际 %d", st.TotalFailures)
	}
	if st.LastFailureAt == nil {
		t.Fatal("last_failure_at 不应为空")
	}
}
