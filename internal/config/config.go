// Package config 负责平台配置的加载与持久化。
// 配置来源优先级: 环境变量 > 配置文件(config.yaml, 由安装程序生成) > 默认值。
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 平台全局配置。
type Config struct {
	App        AppConfig        `yaml:"app"`
	Database   DatabaseConfig   `yaml:"database"`
	Queue      QueueConfig      `yaml:"queue"`
	Store      StoreConfig      `yaml:"store"`
	Auth       AuthConfig       `yaml:"auth"`
	Admin      AdminConfig      `yaml:"admin"`
	RateLimit  RateLimitConfig  `yaml:"ratelimit"`
	Retention  RetentionConfig  `yaml:"retention"`
	Epay       EpayConfig       `yaml:"epay"`
	Server     ServerConfig     `yaml:"server"`
	Redis      RedisConfig      `yaml:"redis"`
	Web        WebConfig        `yaml:"web"`
}

// WebConfig 前端托管配置(用户中心/管理后台端口固化)。
type WebConfig struct {
	UserDist  string `yaml:"user_dist"`  // 用户中心构建产物目录
	AdminDist string `yaml:"admin_dist"` // 管理员后台构建产物目录(base 为 /{admin_path}/)
	UserPort  int    `yaml:"user_port"`  // 用户中心端口(默认 19876)
	AdminPort int    `yaml:"admin_port"` // 管理后台端口(默认 19877)
	APITarget string `yaml:"api_target"` // cmd/web 的 API 反代目标(默认 http://127.0.0.1:{server.port})
}

// AppConfig 应用基础配置。
type AppConfig struct {
	Name        string `yaml:"name"`
	Env         string `yaml:"env"` // local | production
	DataDir     string `yaml:"data_dir"`
	BaseURL     string `yaml:"base_url"`
	UserWebURL  string `yaml:"user_web_url"`
	AdminWebURL string `yaml:"admin_web_url"`
}

// ServerConfig HTTP 服务配置。
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// DatabaseConfig 业务数据库配置(生产 PG / 本地 SQLite)。
type DatabaseConfig struct {
	Type       string `yaml:"type"` // postgres | sqlite
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	User       string `yaml:"user"`
	Password   string `yaml:"password"`
	Name       string `yaml:"name"`
	SSLMode    string `yaml:"sslmode"`
	SQLitePath string `yaml:"sqlite_path"`
}

// QueueConfig 消息队列配置(生产 Kafka / 本地 inprocess)。
type QueueConfig struct {
	Type    string   `yaml:"type"` // kafka | inprocess
	Brokers []string `yaml:"brokers"`
	Topic   string   `yaml:"topic"`
	GroupID string   `yaml:"group_id"`
	// 进程内队列缓冲大小
	BufferSize int `yaml:"buffer_size"`
	// 消费者并发数
	Concurrency int `yaml:"concurrency"`
}

// StoreConfig 消息记录存储配置(生产 ClickHouse / 本地 sqlite)。
type StoreConfig struct {
	Type    string `yaml:"type"` // clickhouse | sqlite
	DSN     string `yaml:"dsn"`
	DBName  string `yaml:"db_name"`
}

// AuthConfig 认证配置。
type AuthConfig struct {
	JWTSecret     string        `yaml:"jwt_secret"`
	TokenTTL      time.Duration `yaml:"token_ttl"`
	APIKeyPrefix  string        `yaml:"api_key_prefix"`
	BcryptCost    int           `yaml:"bcrypt_cost"`
}

// AdminConfig 管理员后台随机路径配置(安全隐藏, 非 /admin)。
type AdminConfig struct {
	// RandomPath 是管理员后台的随机前缀, 由安装程序生成, 可轮换。
	RandomPath string `yaml:"random_path"`
}

// RateLimitConfig 限流配置。
type RateLimitConfig struct {
	// PerMinute 每租户每分钟允许的最大消息受理数。
	PerMinute int `yaml:"per_minute"`
	// PerTargetPerHour 同一收件目标每小时最大消息数(防刷)。
	PerTargetPerHour int `yaml:"per_target_per_hour"`
	// Enabled 是否启用限流。
	Enabled bool `yaml:"enabled"`
}

// RetentionConfig 消息保留策略。
type RetentionConfig struct {
	// MessageDays 消息记录保留天数, 到期由归档任务清理。
	MessageDays int `yaml:"message_days"`
}

// EpayConfig 易支付配置(标准协议, 后台可配, 安装时可留空)。
type EpayConfig struct {
	Gateway string `yaml:"gateway"`
	PID     string `yaml:"pid"`
	Key     string `yaml:"key"`
}

// RedisConfig Redis 单机配置(可选)。
// 留空 Addr = 纯内存模式(限流/熔断用进程内实现)。
type RedisConfig struct {
	Addr             string `yaml:"addr"`   // 127.0.0.1:6379
	Password         string `yaml:"password"`
	DB               int    `yaml:"db"`
	PoolSize         int    `yaml:"pool_size"`
	FallbackInMemory *bool  `yaml:"fallback_inmemory"` // Redis 不可用时降级内存(默认 true)
}

// FilePath 配置文件的默认路径。
const FilePath = "config.yaml"

// Load 加载配置: 先读配置文件, 再用环境变量覆盖, 最后套默认值。
// 配置文件不存在时不视为错误(安装前状态)。
func Load() (*Config, error) {
	cfg := defaultConfig()
	if data, err := os.ReadFile(FilePath); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("解析配置文件失败: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	cfg.applyEnv()
	cfg.applyDefaults()
	return cfg, nil
}

// Save 将配置写入文件(安装程序调用)。
func (c *Config) Save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(FilePath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(FilePath, data, 0o644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	return nil
}

// IsInstalled 判断平台是否已安装(install.lock 存在)。
func IsInstalled() bool {
	_, err := os.Stat(lockPath())
	return err == nil
}

// MarkInstalled 写入安装锁文件。
func MarkInstalled() error {
	f, err := os.Create(lockPath())
	if err != nil {
		return err
	}
	_, err = f.WriteString(time.Now().Format(time.RFC3339) + "\n")
	f.Close()
	return err
}

// lockPath 返回安装锁文件路径。
func lockPath() string {
	return filepath.Join(lockDir(), "install.lock")
}

// lockDir 返回安装锁目录(与配置文件同目录)。
func lockDir() string {
	return filepath.Dir(FilePath)
}

func defaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Name:    "messagepusher",
			Env:     "local",
			DataDir: "./data",
		},
		Server: ServerConfig{Host: "0.0.0.0", Port: 8080},
		Database: DatabaseConfig{
			Type:       "sqlite",
			SQLitePath: "./data/messagepusher.db",
			SSLMode:    "disable",
		},
		Queue: QueueConfig{
			Type:        "inprocess",
			Topic:       "messages",
			GroupID:     "worker",
			BufferSize:  1024,
			Concurrency: 4,
		},
		Store: StoreConfig{Type: "sqlite"},
		Auth: AuthConfig{
			TokenTTL:     24 * time.Hour,
			APIKeyPrefix: "mp_",
			BcryptCost:   10,
		},
		RateLimit: RateLimitConfig{PerMinute: 600, PerTargetPerHour: 20, Enabled: true},
		Retention: RetentionConfig{MessageDays: 90},
		Redis: RedisConfig{
			PoolSize: 20,
		},
	}
}

// applyDefaults 对零值字段套默认值。
func (c *Config) applyDefaults() {
	d := defaultConfig()
	if c.App.Name == "" {
		c.App.Name = d.App.Name
	}
	if c.App.Env == "" {
		c.App.Env = d.App.Env
	}
	if c.App.DataDir == "" {
		c.App.DataDir = d.App.DataDir
	}
	if c.Server.Port == 0 {
		c.Server.Port = d.Server.Port
	}
	if c.Database.Type == "" {
		c.Database.Type = d.Database.Type
	}
	if c.Database.Type == "sqlite" && c.Database.SQLitePath == "" {
		c.Database.SQLitePath = d.Database.SQLitePath
	}
	if c.Queue.Type == "" {
		c.Queue.Type = d.Queue.Type
	}
	if c.Queue.BufferSize <= 0 {
		c.Queue.BufferSize = d.Queue.BufferSize
	}
	if c.Queue.Concurrency <= 0 {
		c.Queue.Concurrency = d.Queue.Concurrency
	}
	if c.Store.Type == "" {
		c.Store.Type = d.Store.Type
	}
	if c.Auth.TokenTTL <= 0 {
		c.Auth.TokenTTL = d.Auth.TokenTTL
	}
	if c.Auth.APIKeyPrefix == "" {
		c.Auth.APIKeyPrefix = d.Auth.APIKeyPrefix
	}
	if c.Auth.BcryptCost <= 0 {
		c.Auth.BcryptCost = d.Auth.BcryptCost
	}
	if c.RateLimit.PerMinute <= 0 {
		c.RateLimit.PerMinute = d.RateLimit.PerMinute
	}
	if c.RateLimit.PerTargetPerHour <= 0 {
		c.RateLimit.PerTargetPerHour = d.RateLimit.PerTargetPerHour
	}
	if c.Retention.MessageDays <= 0 {
		c.Retention.MessageDays = d.Retention.MessageDays
	}
	if c.Redis.PoolSize <= 0 {
		c.Redis.PoolSize = d.Redis.PoolSize
	}
	if c.Redis.FallbackInMemory == nil {
		t := true
		c.Redis.FallbackInMemory = &t
	}
	// Web 端口固化: 用户中心 19876 / 管理后台 19877。
	if c.Web.UserPort <= 0 {
		c.Web.UserPort = 19876
	}
	if c.Web.AdminPort <= 0 {
		c.Web.AdminPort = 19877
	}
	if c.Web.APITarget == "" {
		c.Web.APITarget = fmt.Sprintf("http://127.0.0.1:%d", c.Server.Port)
	}
}

// applyEnv 用环境变量覆盖配置(部署优先)。
func (c *Config) applyEnv() {
	if v := os.Getenv("MP_ENV"); v != "" {
		c.App.Env = v
	}
	if v := os.Getenv("MP_DB_TYPE"); v != "" {
		c.Database.Type = v
	}
	if v := os.Getenv("MP_DB_HOST"); v != "" {
		c.Database.Host = v
	}
	if v := os.Getenv("MP_DB_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &c.Database.Port)
	}
	if v := os.Getenv("MP_DB_USER"); v != "" {
		c.Database.User = v
	}
	if v := os.Getenv("MP_DB_PASSWORD"); v != "" {
		c.Database.Password = v
	}
	if v := os.Getenv("MP_DB_NAME"); v != "" {
		c.Database.Name = v
	}
	if v := os.Getenv("MP_SQLITE_PATH"); v != "" {
		c.Database.SQLitePath = v
	}
	if v := os.Getenv("MP_QUEUE_TYPE"); v != "" {
		c.Queue.Type = v
	}
	if v := os.Getenv("MP_KAFKA_BROKERS"); v != "" {
		c.Queue.Brokers = splitCSV(v)
	}
	if v := os.Getenv("MP_STORE_TYPE"); v != "" {
		c.Store.Type = v
	}
	if v := os.Getenv("MP_CH_DSN"); v != "" {
		c.Store.DSN = v
	}
	if v := os.Getenv("MP_JWT_SECRET"); v != "" {
		c.Auth.JWTSecret = v
	}
	if v := os.Getenv("MP_ADMIN_PATH"); v != "" {
		c.Admin.RandomPath = v
	}
	if v := os.Getenv("MP_REDIS_ADDR"); v != "" {
		c.Redis.Addr = v
	}
	if v := os.Getenv("MP_REDIS_PASSWORD"); v != "" {
		c.Redis.Password = v
	}
	if v := os.Getenv("MP_REDIS_DB"); v != "" {
		fmt.Sscanf(v, "%d", &c.Redis.DB)
	}
	if v := os.Getenv("MP_REDIS_FALLBACK"); v != "" {
		b := v == "1" || v == "true"
		c.Redis.FallbackInMemory = &b
	}
	if v := os.Getenv("MP_USER_DIST"); v != "" {
		c.Web.UserDist = v
	}
	if v := os.Getenv("MP_ADMIN_DIST"); v != "" {
		c.Web.AdminDist = v
	}
	if v := os.Getenv("MP_USER_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &c.Web.UserPort)
	}
	if v := os.Getenv("MP_ADMIN_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &c.Web.AdminPort)
	}
	if v := os.Getenv("MP_API_TARGET"); v != "" {
		c.Web.APITarget = v
	}
	if v := os.Getenv("MP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &c.Server.Port)
	}
}

func splitCSV(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// DSN 返回 GORM 使用的数据库连接串。
func (c *Config) DSN() (string, error) {
	switch c.Database.Type {
	case "sqlite":
		return c.Database.SQLitePath, nil
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Database.Host, c.Database.Port, c.Database.User, c.Database.Password, c.Database.Name, c.Database.SSLMode), nil
	default:
		return "", fmt.Errorf("不支持的数据库类型: %s", c.Database.Type)
	}
}
