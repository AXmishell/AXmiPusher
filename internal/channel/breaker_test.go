package channel

import (
	"testing"
	"time"
)

// TestCircuitBreaker 熔断状态机: 连续失败→OPEN→冷却→HALF_OPEN 探针→成功→CLOSED。
func TestCircuitBreaker(t *testing.T) {
	b := NewCircuitBreaker(3, 50*time.Millisecond)
	const tid = uint64(1)

	// 闭合: 放行, 失败 3 次后熔断。
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
	b.Failure(tid, "email") // 第 3 次
	if !b.IsOpen(tid, "email") {
		t.Fatal("3 次失败后应熔断")
	}
	if b.Allow(tid, "email") {
		t.Fatal("熔断期内应拒绝")
	}

	// 冷却结束 → 半开放行探针。
	time.Sleep(70 * time.Millisecond)
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
	time.Sleep(70 * time.Millisecond)
	b.Allow(tid, "email")
	b.Failure(tid, "email") // 半开探针失败
	if !b.IsOpen(tid, "email") {
		t.Fatal("半开探针失败后应立即重新熔断")
	}

	// 租户隔离: 租户 2 的 email 不受影响。
	if b.IsOpen(2, "email") {
		t.Fatal("熔断应按租户隔离")
	}
}
