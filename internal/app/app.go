// Package app 应用容器: 装配配置、数据库、存储、队列与全部服务。
// api 与 worker 两个进程共用此容器, 本地模式下 api 内嵌消费者。
package app

import (
	"context"
	"fmt"
	"time"

	"messagepusher/internal/channel"
	"messagepusher/internal/config"
	"messagepusher/internal/db"
	"messagepusher/internal/queue"
	"messagepusher/internal/service"
	"messagepusher/internal/store"
	"messagepusher/internal/worker"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// App 应用容器。
type App struct {
	Cfg       *config.Config
	DB        *gorm.DB
	Store     store.MessageStore
	Queue     queue.Queue
	Auth      *service.AuthService
	Messages  *service.MessageService
	Registry  *channel.Registry
	Limiter   service.RateLimiter
	Templates *service.TemplateService
	Settings  *service.SettingsService
	Pay       *service.PaymentService
	Batch     *service.BatchService
	Redis     *redis.Client
	// RedisMode 是否使用 Redis 分布式实现(限流/熔断)。
	RedisMode bool
	// Router 由 api 进程注入, 供轮换 admin 路径时动态注册新前缀路由。
	Router interface {
		GET(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	}
}

// New 创建并构建应用。
func New(cfg *config.Config) (*App, error) {
	a := &App{Cfg: cfg}
	if err := a.Build(); err != nil {
		return nil, err
	}
	return a, nil
}

// Build 根据当前配置构建全部组件。
func (a *App) Build() error {
	cfg := a.Cfg

	// 业务数据库。
	gdb, err := db.Open(cfg)
	if err != nil {
		return err
	}
	a.DB = gdb

	// 消息记录存储。
	switch cfg.Store.Type {
	case "sqlite":
		st, err := store.NewSQLiteStore(gdb)
		if err != nil {
			return err
		}
		a.Store = st
	default:
		return fmt.Errorf("不支持的存储类型: %s", cfg.Store.Type)
	}

	// 消息队列。
	switch cfg.Queue.Type {
	default:
		return fmt.Errorf("不支持的队列类型: %s", cfg.Queue.Type)
	}

	// 服务。
	a.Auth = service.NewAuthService(gdb, cfg)
	a.Limiter = service.NewMemoryRateLimiter(cfg.RateLimit.PerMinute)
	a.Messages = service.NewMessageService(gdb, a.Store, a.Queue, a.Limiter, cfg.RateLimit.Enabled)
	a.Templates = service.NewTemplateService(gdb)
	a.Settings = service.NewSettingsService(gdb)
	a.Pay = service.NewPaymentService(gdb, a.Settings, cfg.App.BaseURL)
	a.Batch = service.NewBatchService(gdb, a.Messages)

	// Redis 单机(可选): 连接探活, 失败按配置降级内存。
	if err := a.setupRedis(); err != nil {
		return err
	}

	// 渠道注册(免费渠道)。
	var breaker channel.Breaker
	if a.RedisMode {
		breaker = channel.NewRedisCircuitBreaker(a.Redis, 3, 30*time.Second)
	} else {
		breaker = channel.NewCircuitBreaker(3, 30*time.Second)
	}
	a.Registry = channel.NewRegistry()
	a.Registry.SetBreaker(breaker)
	a.Registry.Register(channel.NewWebhookSender(gdb))
	a.Registry.Register(channel.NewEmailSender(gdb, a.Settings))
	a.Registry.Register(channel.NewAPNsSender(gdb))
	a.Registry.Register(channel.NewFCMSender(gdb))
	a.Registry.Register(channel.NewInAppSender(gdb))
	// 消息服务依赖渠道注册表做快速失败校验 + 熔断降级路由。
	a.Messages.HasChannel = a.Registry.HasChannel
	a.Messages.IsAvailable = a.Registry.IsAvailable

	return nil
}

// setupRedis 连接 Redis, 探活失败按配置降级。
func (a *App) setupRedis() error {
	cfg := a.Cfg.Redis
	if cfg.Addr == "" {
		return nil // 未配置: 纯内存模式。
	}
	fallback := true
	if cfg.FallbackInMemory != nil {
		fallback = *cfg.FallbackInMemory
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		if fallback {
			fmt.Printf("[redis] %s 不可用, 已降级内存模式: %v\n", cfg.Addr, err)
			return nil
		}
		return fmt.Errorf("Redis 不可用且未开启降级: %w", err)
	}
	a.Redis = rdb
	a.RedisMode = true

	// 限流/幂等切换到 Redis 实现。
	a.Limiter = service.NewRedisRateLimiter(rdb, a.Cfg.RateLimit.PerMinute)
	a.Messages.SetLimiter(a.Limiter)
	a.Messages.SetRedisCache(rdb)
	fmt.Printf("[redis] 已连接 %s, 分布式限流/熔断/幂等加速启用\n", cfg.Addr)
	return nil
}

// Reinit 用新配置重建全部组件(安装程序写入配置后调用)。
// 注意: 同类型队列会保留 —— 消费者已订阅旧队列实例, 重建会导致消息无人消费。
func (a *App) Reinit(cfg *config.Config) error {
	var keepQueue queue.Queue
	if a.Queue != nil && a.Queue.Type() == cfg.Queue.Type {
		keepQueue = a.Queue
	}
	if a.Store != nil {
		a.Store.Close()
	}
	a.Cfg = cfg
	if err := a.Build(); err != nil {
		return err
	}
	if keepQueue != nil {
		if a.Queue != nil {
			a.Queue.Close()
		}
		a.Queue = keepQueue
		a.Messages.SetQueue(keepQueue)
	}
	return nil
}

// StartConsumer 启动消费者循环(本地模式由 api 进程调用; 生产模式由 worker 进程调用)。
// 阻塞直到 ctx 取消。worker 每次消费动态取最新 store/registry(Reinit 后生效)。
func (a *App) StartConsumer(ctx context.Context) error {
	return a.Queue.Subscribe(ctx, func(c context.Context, msg *queue.TaskMessage) error {
		w := worker.New(func() store.MessageStore { return a.Store }, func() *channel.Registry { return a.Registry })
		w.Handle(c, msg)
		return nil
	})
}

// Close 释放资源。
func (a *App) Close() {
	if a.Queue != nil {
		a.Queue.Close()
	}
	if a.Store != nil {
		a.Store.Close()
	}
	if a.Redis != nil {
		a.Redis.Close()
	}
}
