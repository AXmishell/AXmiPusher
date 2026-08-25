package service

import (
	"context"
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

// TestRedisRateLimiter 固定窗口限流: 限额内放行, 超限拒绝, 窗口重置恢复。
func TestRedisRateLimiter(t *testing.T) {
	mr, rdb := newTestRedis(t)
	limiter := NewRedisRateLimiter(rdb, 3)

	// 额度 3: 前 3 次放行。
	for i := 0; i < 3; i++ {
		if !limiter.Allow(1) {
			t.Fatalf("第 %d 次应在额度内放行", i+1)
		}
	}
	// 第 4 次拒绝。
	if limiter.Allow(1) {
		t.Fatal("超限应拒绝")
	}
	// 租户隔离。
	if !limiter.Allow(2) {
		t.Fatal("租户 2 不受租户 1 限额影响")
	}
	// 窗口重置: 快进 61 秒后恢复。
	mr.FastForward(61 * time.Second)
	for i := 0; i < 3; i++ {
		if !limiter.Allow(1) {
			t.Fatalf("窗口重置后第 %d 次应放行", i+1)
		}
	}
	if limiter.Allow(1) {
		t.Fatal("重置后超限应拒绝")
	}
}

// TestRedisIdempotencyScripts 幂等缓存脚本 GET/SET 行为。
func TestRedisIdempotencyScripts(t *testing.T) {
	_, rdb := newTestRedis(t)
	key := "idem:1:req-x"

	// 未设置: GET 返回 0。
	v, err := rdb.Eval(context.Background(), redisGetScript, []string{key}).Int64()
	if err != nil || v != 0 {
		t.Fatalf("空 key GET 应返回 0, got %d err=%v", v, err)
	}
	// 设置。
	if _, err := rdb.Eval(context.Background(), redisSetScript, []string{key}, 42, 86400).Result(); err != nil {
		t.Fatal(err)
	}
	// 命中。
	v, err = rdb.Eval(context.Background(), redisGetScript, []string{key}).Int64()
	if err != nil || v != 42 {
		t.Fatalf("SET 后 GET 应返回 42, got %d err=%v", v, err)
	}
}
