// API 服务入口: 提供核心 API + 兼容层 + 安装向导, 同时内嵌数据库轮询消费者。
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"axmipusher/internal/api"
	"axmipusher/internal/app"
	"axmipusher/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	a, err := app.New(cfg)
	if err != nil {
		log.Fatalf("初始化应用失败: %v", err)
	}
	defer a.Close()

	api.SetInstalledChecker(config.IsInstalled)
	router := api.NewRouter(a)
	a.Router = router

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 数据库轮询消费者(单一队列实现, api 进程恒消费)。
	go func() {
		log.Printf("启动数据库轮询消费者(轮询间隔=%v 批量=%d 并发=%d)", cfg.Queue.PollInterval, cfg.Queue.BatchSize, cfg.Queue.Concurrency)
		if err := a.StartConsumer(ctx); err != nil {
			log.Printf("消费者退出: %v", err)
		}
	}()

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("AXmiPusher API 已启动: http://%s", addr)
		if cfg.Admin.RandomPath != "" {
			log.Printf("管理员后台路径: /%s/", cfg.Admin.RandomPath)
		}
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP 服务异常: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("正在关闭服务...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}
