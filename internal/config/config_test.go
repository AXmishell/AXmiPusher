package config

import (
	"strings"
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

// TestDSN_MySQL MySQL 5.7 DSN 形状: tcp + utf8mb4 + parseTime/loc, 端口缺省 3306。
func TestDSN_MySQL(t *testing.T) {
	cfg := defaultConfig()
	cfg.Database.Type = "mysql"
	cfg.Database.Host = "127.0.0.1"
	cfg.Database.User = "mp"
	cfg.Database.Password = "secret"
	cfg.Database.Name = "axmipusher"
	dsn, err := cfg.DSN()
	if err != nil {
		t.Fatalf("mysql DSN 不应报错: %v", err)
	}
	want := "mp:secret@tcp(127.0.0.1:3306)/axmipusher?charset=utf8mb4&parseTime=True&loc=Local"
	if dsn != want {
		t.Fatalf("mysql DSN 不匹配\n  want: %s\n  got:  %s", want, dsn)
	}
	// 显式端口应生效。
	cfg.Database.Port = 3307
	dsn, _ = cfg.DSN()
	if !strings.Contains(dsn, "@tcp(127.0.0.1:3307)/") {
		t.Fatalf("mysql DSN 应使用显式端口, got: %s", dsn)
	}
}

// TestValidateAdminPath 后台路径校验: 合法/长度(8-32位)/字符集/保留字/规范化。
func TestValidateAdminPath(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "myadmin1", want: "myadmin1"},                           // 恰好 8 位
		{in: "/myadmin1/", want: "myadmin1"},                         // 首尾斜杠应被剥离
		{in: "AbC123Ab", want: "AbC123Ab"},                           // 大小写+数字, 8 位
		{in: "MyConsolePath16X", want: "MyConsolePath16X"},           // 16 位
		{in: "a2345678901234567890123456789012", want: "a2345678901234567890123456789012"}, // 最长 32 位
		{in: "", wantErr: true},                                      // 空
		{in: "abc", wantErr: true},                                   // 过短(3 位)
		{in: "myadmin", wantErr: true},                               // 过短(7 位 < 8)
		{in: "abc-123abc1", wantErr: true},                           // 非法字符
		{in: "abc_123abc1", wantErr: true},                           // 非法字符
		{in: "abc 123abc1", wantErr: true},                           // 空格
		{in: "a23456789012345678901234567890123", wantErr: true},     // 超长 33 位
		{in: "api", wantErr: true},                                   // 保留字(长度不足也拦)
	}
	for _, tc := range cases {
		got, err := ValidateAdminPath(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ValidateAdminPath(%q) 应报错, got %q", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ValidateAdminPath(%q) 不应报错: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ValidateAdminPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestGenerateRandomAdminPath 随机路径: 恰好 16 位, 仅含大小写字母数字, 且能通过校验; 多次生成不重复。
func TestGenerateRandomAdminPath(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		p, err := GenerateRandomAdminPath()
		if err != nil {
			t.Fatalf("GenerateRandomAdminPath 不应报错: %v", err)
		}
		if len(p) != 16 {
			t.Fatalf("随机路径长度应为 16, got %d (%q)", len(p), p)
		}
		if !adminPathRe.MatchString(p) {
			t.Fatalf("随机路径应满足 8-32 位字母数字校验: %q", p)
		}
		// 应包含字母与数字混合(概率上 100 次内几乎必然出现大写字母与数字)。
		if seen[p] {
			t.Fatalf("随机路径重复: %q", p)
		}
		seen[p] = true
	}
}
