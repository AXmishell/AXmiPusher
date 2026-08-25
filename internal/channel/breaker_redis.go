package channel

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCircuitBreaker Redis 熔断器(多实例共享状态)。
// 状态存储: breaker:{tenant}:{channel} HASH
//   state=0闭合 1熔断 2半开; failures; opened_at; half_at; total_failures; total_successes; last_success_at; last_failure_at
// 所有状态转移用 Lua 原子脚本, 保证多实例并发正确。
type RedisCircuitBreaker struct {
	client    *redis.Client
	threshold int
	cooldown  time.Duration
}

// NewRedisCircuitBreaker 创建 Redis 熔断器。
func NewRedisCircuitBreaker(client *redis.Client, threshold int, cooldown time.Duration) *RedisCircuitBreaker {
	if threshold <= 0 {
		threshold = 3
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &RedisCircuitBreaker{client: client, threshold: threshold, cooldown: cooldown}
}

// ensure 接口实现。
var _ Breaker = (*RedisCircuitBreaker)(nil)

func (b *RedisCircuitBreaker) key(tenantID uint64, channel string) string {
	return fmt.Sprintf("breaker:%d:%s", tenantID, channel)
}

// breakerAllowScript 放行判定: 读状态 → 决定是否放行/转移状态。
const breakerAllowScript = `
local h = redis.call('HGETALL', KEYS[1])
local state, opened_at, half_at = 0, 0, 0
for i=1,#h,2 do
    if h[i] == 'state' then state = tonumber(h[i+1])
    elseif h[i] == 'opened_at' then opened_at = tonumber(h[i+1])
    elseif h[i] == 'half_at' then half_at = tonumber(h[i+1]) end
end
if #h == 0 then
    redis.call('HSET', KEYS[1], 'state', 0, 'failures', 0, 'opened_at', 0, 'half_at', 0, 'total_failures', 0, 'total_successes', 0, 'last_success_at', 0, 'last_failure_at', 0)
    redis.call('PEXPIRE', KEYS[1], ARGV[3])
    return 1
end
local now = tonumber(ARGV[1])
local cooldown_ms = tonumber(ARGV[2])
if state == 1 then
    if now - opened_at >= cooldown_ms then
        redis.call('HSET', KEYS[1], 'state', 2, 'half_at', now)
        return 1
    end
    return 0
end
if state == 2 then
    if now - half_at >= cooldown_ms then
        redis.call('HSET', KEYS[1], 'half_at', now)
        return 1
    end
    return 0
end
return 1
`

// Allow 判断是否放行。
func (b *RedisCircuitBreaker) Allow(tenantID uint64, channel string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := b.client.Eval(ctx, breakerAllowScript, []string{b.key(tenantID, channel)},
		time.Now().UnixMilli(), b.cooldown.Milliseconds(), (b.cooldown * 10).Milliseconds()).Result()
	if err != nil {
		// Redis 异常: 放行(容错优先)。
		return true
	}
	return res.(int64) == 1
}

// breakerSuccessScript 成功: 重置为闭合。
const breakerSuccessScript = `
redis.call('HSET', KEYS[1], 'state', 0, 'failures', 0, 'last_success_at', ARGV[1])
redis.call('HINCRBY', KEYS[1], 'total_successes', 1)
return 1
`

// Success 调用成功。
func (b *RedisCircuitBreaker) Success(tenantID uint64, channel string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	b.client.Eval(ctx, breakerSuccessScript, []string{b.key(tenantID, channel)}, time.Now().UnixMilli())
}

// breakerFailureScript 失败: 计数, 达阈值或半开探针失败则熔断。
const breakerFailureScript = `
local h = redis.call('HGETALL', KEYS[1])
local state, failures = 0, 0
for i=1,#h,2 do
    if h[i] == 'state' then state = tonumber(h[i+1])
    elseif h[i] == 'failures' then failures = tonumber(h[i+1]) end
end
local now = tonumber(ARGV[1])
local threshold = tonumber(ARGV[2])
local new_failures = failures + 1
redis.call('HSET', KEYS[1], 'failures', new_failures, 'last_failure_at', now)
redis.call('HINCRBY', KEYS[1], 'total_failures', 1)
if state == 2 or new_failures >= threshold then
    redis.call('HSET', KEYS[1], 'state', 1, 'opened_at', now)
end
return 1
`

// Failure 调用失败。
func (b *RedisCircuitBreaker) Failure(tenantID uint64, channel string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	b.client.Eval(ctx, breakerFailureScript, []string{b.key(tenantID, channel)},
		time.Now().UnixMilli(), b.threshold)
}

// breakerIsOpenScript 是否处于熔断(冷却期内)。
const breakerIsOpenScript = `
local h = redis.call('HGETALL', KEYS[1])
local state, opened_at = 0, 0
for i=1,#h,2 do
    if h[i] == 'state' then state = tonumber(h[i+1])
    elseif h[i] == 'opened_at' then opened_at = tonumber(h[i+1]) end
end
if state == 1 then
    local now = tonumber(ARGV[1])
    local cooldown_ms = tonumber(ARGV[2])
    if now - opened_at < cooldown_ms then return 1 end
end
return 0
`

// IsOpen 渠道是否熔断。
func (b *RedisCircuitBreaker) IsOpen(tenantID uint64, channel string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := b.client.Eval(ctx, breakerIsOpenScript, []string{b.key(tenantID, channel)},
		time.Now().UnixMilli(), b.cooldown.Milliseconds()).Result()
	if err != nil {
		return false
	}
	return res.(int64) == 1
}

// breakerStatsScript 观测数据。
const breakerStatsScript = `
local h = redis.call('HGETALL', KEYS[1])
if #h == 0 then return {0,0,0,0,0,0} end
local m = {}
for i=1,#h,2 do m[h[i]] = h[i+1] end
return {m['state'] or 0, m['failures'] or 0, m['total_failures'] or 0, m['total_successes'] or 0, m['last_success_at'] or 0, m['last_failure_at'] or 0}
`

// Stats 熔断观测数据。
func (b *RedisCircuitBreaker) Stats(tenantID uint64, channel string) BreakerStats {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := b.client.Eval(ctx, breakerStatsScript, []string{b.key(tenantID, channel)}).Result()
	if err != nil {
		return BreakerStats{State: "closed"}
	}
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 4 {
		return BreakerStats{State: "closed"}
	}
	state := map[int64]string{0: "closed", 1: "open", 2: "half_open"}[toInt64(arr[0])]
	stats := BreakerStats{
		State:          state,
		Failures:       int(toInt64(arr[1])),
		TotalFailures:  toInt64(arr[2]),
		TotalSuccesses: toInt64(arr[3]),
	}
	if len(arr) >= 6 {
		if v := toInt64(arr[4]); v > 0 {
			t := time.UnixMilli(v)
			stats.LastSuccessAt = &t
		}
		if v := toInt64(arr[5]); v > 0 {
			t := time.UnixMilli(v)
			stats.LastFailureAt = &t
		}
	}
	return stats
}

func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case string:
		i, _ := strconv.ParseInt(n, 10, 64)
		return i
	}
	return 0
}
