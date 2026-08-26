package service

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter 限流器接口。
// Redis 模式: 多实例共享窗口计数(分布式正确); 内存模式: 单实例。
type RateLimiter interface {
	// Allow 判断该租户是否允许通过(消耗一个额度)。
	Allow(tenantID uint64) bool
	// Type 实现类型(memory | redis)。
	Type() string
	// SetPerMinute 动态调整每租户每分钟额度(管理后台设置生效)。
	SetPerMinute(n int)
}

// MemoryRateLimiter 内存令牌桶限流器(单实例 / 降级)。
type MemoryRateLimiter struct {
	mu        sync.Mutex
	buckets   map[uint64]*tokenBucket
	perMinute int
}

type tokenBucket struct {
	tokens float64
	last   time.Time
}

// NewMemoryRateLimiter 创建内存限流器, perMinute 为每租户每分钟额度。
func NewMemoryRateLimiter(perMinute int) *MemoryRateLimiter {
	if perMinute <= 0 {
		perMinute = 600
	}
	return &MemoryRateLimiter{
		buckets:   make(map[uint64]*tokenBucket),
		perMinute: perMinute,
	}
}

// Allow 判断该租户是否允许通过(消耗一个令牌)。
func (r *MemoryRateLimiter) Allow(tenantID uint64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, ok := r.buckets[tenantID]
	if !ok {
		r.buckets[tenantID] = &tokenBucket{tokens: float64(r.perMinute), last: time.Now()}
		return true
	}
	// 按时间补充令牌。
	now := time.Now()
	elapsed := now.Sub(b.last).Minutes()
	b.tokens += elapsed * float64(r.perMinute)
	if b.tokens > float64(r.perMinute) {
		b.tokens = float64(r.perMinute)
	}
	b.last = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// Type 返回类型。
func (r *MemoryRateLimiter) Type() string { return "memory" }

// SetPerMinute 动态调整额度。
func (r *MemoryRateLimiter) SetPerMinute(n int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n > 0 {
		r.perMinute = n
	}
}

// redisRateLimitScript 固定窗口计数(INCR + PEXPIRE), 原子且自动续期。
const redisRateLimitScript = `
local current = redis.call('INCR', KEYS[1])
if current == 1 then
    redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return current <= tonumber(ARGV[2]) and 1 or 0
`

// redisCmdable 限定最小命令集(便于测试注入, 与 go-redis 签名一致)。
type redisCmdable interface {
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
}

// RedisRateLimiter Redis 固定窗口限流器(分布式)。
// 窗口: 每分钟; 每租户独立 key: ratelimit:{tenant}。
type RedisRateLimiter struct {
	client    redisCmdable
	perMinute atomic.Int64
}

// NewRedisRateLimiter 创建 Redis 限流器。
func NewRedisRateLimiter(client redisCmdable, perMinute int) *RedisRateLimiter {
	if perMinute <= 0 {
		perMinute = 600
	}
	r := &RedisRateLimiter{client: client}
	r.perMinute.Store(int64(perMinute))
	return r
}

// Allow 使用 Lua 原子脚本判断限额。
func (r *RedisRateLimiter) Allow(tenantID uint64) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	key := "ratelimit:" + strconv.FormatUint(tenantID, 10)
	res, err := r.client.Eval(ctx, redisRateLimitScript, []string{key}, 60000, r.perMinute.Load()).Int64()
	if err != nil {
		// Redis 异常: 放行(容错优先, 由审计兜底)。
		return true
	}
	return res == 1
}

// Type 返回类型。
func (r *RedisRateLimiter) Type() string { return "redis" }

// SetPerMinute 动态调整额度。
func (r *RedisRateLimiter) SetPerMinute(n int) {
	if n > 0 {
		r.perMinute.Store(int64(n))
	}
}
