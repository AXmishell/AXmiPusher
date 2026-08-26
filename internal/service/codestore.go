package service

import (
	"context"
	"sync"
	"time"
)

// CodeStore 一次性验证码存储(注册邮箱验证码等)。
// Redis 模式: 多实例共享(分布式正确); 内存模式: 单实例。
type CodeStore interface {
	// Set 写入验证码(覆盖写, 自带过期时间)。
	Set(key, code string, ttl time.Duration) error
	// Consume 校验并消费验证码: 匹配则删除返回 true, 否则返回 false(一次性)。
	Consume(key, code string) (bool, error)
	// Exists 判断 key 对应的验证码是否存在且未过期。
	Exists(key string) (bool, error)
}

// memCode 内存验证码项。
type memCode struct {
	code string
	exp  time.Time
}

// MemoryCodeStore 内存验证码存储(单实例 / Redis 降级)。
type MemoryCodeStore struct {
	mu    sync.Mutex
	codes map[string]memCode
}

// NewMemoryCodeStore 创建内存验证码存储。
func NewMemoryCodeStore() *MemoryCodeStore {
	return &MemoryCodeStore{codes: make(map[string]memCode)}
}

// Set 写入验证码(覆盖写)。
func (s *MemoryCodeStore) Set(key, code string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[key] = memCode{code: code, exp: time.Now().Add(ttl)}
	return nil
}

// Consume 校验并消费验证码: 存在即删除(一次性), 匹配才返回 true。
func (s *MemoryCodeStore) Consume(key, code string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	mc, ok := s.codes[key]
	if !ok {
		return false, nil
	}
	// 一次性: 无论是否过期/匹配, 消费后即删除。
	delete(s.codes, key)
	if time.Now().After(mc.exp) {
		return false, nil
	}
	return mc.code == code, nil
}

// Exists 判断验证码是否存在且未过期(惰性清理过期项)。
func (s *MemoryCodeStore) Exists(key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	mc, ok := s.codes[key]
	if !ok {
		return false, nil
	}
	if time.Now().After(mc.exp) {
		delete(s.codes, key) // 惰性清理过期项。
		return false, nil
	}
	return true, nil
}

// consumeCodeScript 校验并消费验证码: GET + DEL 原子, 匹配才返回 1。
const consumeCodeScript = `
local v = redis.call('GET', KEYS[1])
if not v then
    return 0
end
redis.call('DEL', KEYS[1])
if v == ARGV[1] then
    return 1
end
return 0
`

// RedisCodeStore Redis 验证码存储(分布式)。
type RedisCodeStore struct {
	client redisCmdable
}

// NewRedisCodeStore 创建 Redis 验证码存储。
func NewRedisCodeStore(client redisCmdable) *RedisCodeStore {
	return &RedisCodeStore{client: client}
}

// Set 写入验证码(覆盖写, 走 SET key value PX ttl)。
func (s *RedisCodeStore) Set(key, code string, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.client.Set(ctx, key, code, ttl).Err()
}

// Consume 用 Lua 原子脚本校验并消费验证码(一次性)。
func (s *RedisCodeStore) Consume(key, code string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := s.client.Eval(ctx, consumeCodeScript, []string{key}, code).Int64()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

// Exists 判断验证码是否存在且未过期。
func (s *RedisCodeStore) Exists(key string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	n, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
