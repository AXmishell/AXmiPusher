package config

import (
	"testing"
)

// TestParseConfig_BackwardCompat 旧配置向后兼容:
// 旧版 config.yaml 含已废弃的本地/生产模式字段(app 环境 / queue 的 Kafka 项 / store 存储),
// 应被忽略不报错, 且新的数据库轮询队列配置默认值生效。
func TestParseConfig_BackwardCompat(t *testing.T) {
	// 清理可能影响断言的环境变量, 确保走纯默认值兜底路径。
	for _, k := range []string{
		"MP_QUEUE_POLL_INTERVAL",
		"MP_QUEUE_BATCH_SIZE",
		"MP_QUEUE_CLAIM_TIMEOUT",
		"MP_QUEUE_MAX_CLAIM_ATTEMPTS",
		"MP_QUEUE_CONCURRENCY",
	} {
		t.Setenv(k, "")
	}

	// Given: 旧版 config.yaml(含已废弃的本地/生产模式字段)。
	data := []byte(`
app:
  env: local
queue:
  type: kafka
  brokers: [a]
  topic: t
  group_id: g
  buffer_size: 1
store:
  type: clickhouse
  dsn: x
`)

	// When: 按配置管道解析(反序列化 → 环境变量 → 默认值兜底)。
	cfg, err := ParseConfig(data)

	// Then: 不报错, 旧字段被忽略, 新轮询配置默认值生效。
	if err != nil {
		t.Fatalf("解析含旧字段的配置不应报错, got: %v", err)
	}
	if cfg.Queue.PollInterval != 500 {
		t.Errorf("PollInterval 应为默认 500, got %d", cfg.Queue.PollInterval)
	}
	if cfg.Queue.BatchSize != 100 {
		t.Errorf("BatchSize 应为默认 100, got %d", cfg.Queue.BatchSize)
	}
	if cfg.Queue.ClaimTimeout != 300 {
		t.Errorf("ClaimTimeout 应为默认 300, got %d", cfg.Queue.ClaimTimeout)
	}
	if cfg.Queue.MaxClaimAttempts != 5 {
		t.Errorf("MaxClaimAttempts 应为默认 5, got %d", cfg.Queue.MaxClaimAttempts)
	}
	if cfg.Queue.Concurrency != 4 {
		t.Errorf("Concurrency 应为默认 4, got %d", cfg.Queue.Concurrency)
	}
}
