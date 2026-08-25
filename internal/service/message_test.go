package service

import (
	"testing"
	"time"

	"messagepusher/internal/channel"
)

// TestAutoFailover 验证 auto 渠道熔断降级逻辑。
func TestAutoFailover(t *testing.T) {
	breaker := channel.NewCircuitBreaker(1, time.Second)
	// 模拟 email 渠道熔断。
	breaker.Failure(1, "email")
	if !breaker.IsOpen(1, "email") {
		t.Fatal("email 应已熔断")
	}

	ms := &MessageService{}
	ms.IsAvailable = func(tid uint64, ch string) bool {
		return !breaker.IsOpen(tid, ch)
	}
	ms.HasChannel = func(ch string) bool { return true }

	if ms.IsAvailable(1, "email") {
		t.Fatal("IsAvailable(email) 应为 false")
	}
	if !ms.IsAvailable(1, "webhook") {
		t.Fatal("IsAvailable(webhook) 应为 true")
	}

	fb := ms.pickFallbackChannel(1, "email")
	if fb != "webhook" {
		t.Fatalf("降级应选 webhook, 实际 %s", fb)
	}

	// 多渠道熔断: email + webhook 都断时, 降级到下一个可用渠道(apns)。
	breaker.Failure(1, "webhook")
	fb2 := ms.pickFallbackChannel(1, "email")
	if fb2 != "apns" {
		t.Fatalf("email/webhook 熔断时应降级 apns, 实际 %s", fb2)
	}

	// 全部渠道熔断时回退到优先渠道(交给 worker 死信)。
	for _, ch := range []string{"apns", "fcm", "inapp"} {
		breaker.Failure(1, ch)
	}
	fb3 := ms.pickFallbackChannel(1, "email")
	if fb3 != "email" {
		t.Fatalf("全部熔断时应回退优先渠道, 实际 %s", fb3)
	}
}
