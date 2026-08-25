// API 服务入口: 提供核心 API + 兼容层 + 安装向导。
// 本地模式下同时内嵌消费者。
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

	"messagepusher/internal/api"
	"messagepusher/internal/app"
	"messagepusher/internal/config"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 本地模式: 进程内启动消费者。
	if cfg.Queue.Type == "inprocess" {
		go func() {
			log.Printf("本地模式: 启动进程内消费者(%d 并发)", cfg.Queue.Concurrency)
			if err := a.StartConsumer(ctx); err != nil {
				log.Printf("消费者退出: %v", err)
			}
		}()
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("MessagePusher API 已启动: http://%s", addr)
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
