package channel

import (
	"fmt"
	"sync"
	"time"
)

// 熔断器状态。
type breakerState int

const (
	stateClosed   breakerState = iota // 正常, 放行
	stateOpen                         // 熔断, 快速失败
	stateHalfOpen                     // 探测, 放行一个探针
)

// ErrCircuitOpen 渠道熔断错误(不重试, 直接死信)。
type ErrCircuitOpen struct {
	Channel string
}

func (e *ErrCircuitOpen) Error() string {
	return fmt.Sprintf("渠道 %s 已熔断", e.Channel)
}

// circuit 单个渠道熔断状态。
type circuit struct {
	state          breakerState
	failures       int
	openedAt       time.Time
	halfAt         time.Time
	lastSuccessAt  time.Time
	lastFailureAt  time.Time
	totalFailures  int64
	totalSuccesses int64
}

// CircuitBreaker 熔断器: 按 (租户, 渠道) 维度维护状态。
// 连续失败超过阈值 → OPEN(冷却期内快速失败); 冷却后 HALF_OPEN 放行探针;
// 探针成功 → CLOSED, 失败 → 重新 OPEN。
type CircuitBreaker struct {
	mu        sync.Mutex
	states    map[string]*circuit
	threshold int           // 连续失败阈值
	cooldown  time.Duration // 熔断冷却期
}

// NewCircuitBreaker 创建熔断器。
func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 3
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &CircuitBreaker{
		states:    make(map[string]*circuit),
		threshold: threshold,
		cooldown:  cooldown,
	}
}

// ensure CircuitBreaker 实现 Breaker 接口。
var _ Breaker = (*CircuitBreaker)(nil)

func (b *CircuitBreaker) key(tenantID uint64, channel string) string {
	return fmt.Sprintf("%d:%s", tenantID, channel)
}

// Allow 判断是否放行本次调用。
func (b *CircuitBreaker) Allow(tenantID uint64, channel string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := b.key(tenantID, channel)
	c, ok := b.states[key]
	if !ok {
		b.states[key] = &circuit{state: stateClosed}
		return true
	}
	now := time.Now()
	switch c.state {
	case stateClosed:
		return true
	case stateOpen:
		if now.Sub(c.openedAt) >= b.cooldown {
			// 冷却结束, 进入半开探测。
			c.state = stateHalfOpen
			c.halfAt = now
			return true
		}
		return false
	case stateHalfOpen:
		// 半开状态只放行一次探针(冷却期内未再触发则放行, 简单实现: 每次冷却后放行一次)。
		if now.Sub(c.halfAt) >= b.cooldown {
			c.halfAt = now
			return true
		}
		return false
	}
	return true
}

// Success 调用成功: 重置计数, 回到闭合。
func (b *CircuitBreaker) Success(tenantID uint64, channel string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := b.key(tenantID, channel)
	if c, ok := b.states[key]; ok {
		c.state = stateClosed
		c.failures = 0
		c.lastSuccessAt = time.Now()
		c.totalSuccesses++
	}
}

// Failure 调用失败: 计数, 达阈值则熔断。
func (b *CircuitBreaker) Failure(tenantID uint64, channel string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := b.key(tenantID, channel)
	c, ok := b.states[key]
	if !ok {
		c = &circuit{}
		b.states[key] = c
	}
	c.failures++
	c.lastFailureAt = time.Now()
	c.totalFailures++
	switch c.state {
	case stateHalfOpen:
		// 探针失败: 立即重新熔断。
		c.state = stateOpen
		c.openedAt = time.Now()
	case stateClosed:
		if c.failures >= b.threshold {
			c.state = stateOpen
			c.openedAt = time.Now()
		}
	}
}

// IsOpen 渠道当前是否熔断(供 auto 路由降级判断)。
func (b *CircuitBreaker) IsOpen(tenantID uint64, channel string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := b.key(tenantID, channel)
	if c, ok := b.states[key]; ok {
		if c.state == stateOpen {
			now := time.Now()
			if now.Sub(c.openedAt) < b.cooldown {
				return true
			}
		}
	}
	return false
}

// Breaker 熔断器接口(memory 与 redis 实现)。
type Breaker interface {
	Allow(tenantID uint64, channel string) bool
	Success(tenantID uint64, channel string)
	Failure(tenantID uint64, channel string)
	IsOpen(tenantID uint64, channel string) bool
	Stats(tenantID uint64, channel string) BreakerStats
}

// BreakerStats 熔断器观测数据(健康看板用)。
type BreakerStats struct {
	State          string    `json:"state"` // closed | open | half_open
	Failures       int       `json:"failures"`
	TotalFailures  int64     `json:"total_failures"`
	TotalSuccesses int64     `json:"total_successes"`
	LastSuccessAt  *time.Time `json:"last_success_at"`
	LastFailureAt  *time.Time `json:"last_failure_at"`
}

// StateName 状态名。
func (s breakerState) String() string {
	switch s {
	case stateOpen:
		return "open"
	case stateHalfOpen:
		return "half_open"
	default:
		return "closed"
	}
}

// Stats 返回熔断器观测数据(不存在则返回闭合初始态)。
func (b *CircuitBreaker) Stats(tenantID uint64, channel string) BreakerStats {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := b.key(tenantID, channel)
	c, ok := b.states[key]
	if !ok {
		return BreakerStats{State: "closed"}
	}
	stats := BreakerStats{
		State:          c.state.String(),
		Failures:       c.failures,
		TotalFailures:  c.totalFailures,
		TotalSuccesses: c.totalSuccesses,
	}
	if !c.lastSuccessAt.IsZero() {
		t := c.lastSuccessAt
		stats.LastSuccessAt = &t
	}
	if !c.lastFailureAt.IsZero() {
		t := c.lastFailureAt
		stats.LastFailureAt = &t
	}
	return stats
}
