// worker 消费者入口: 生产模式下独立进程消费队列并发送。
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"messagepusher/internal/app"
	"messagepusher/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Queue.Type == "inprocess" {
		log.Fatalf("进程内队列不需要独立 worker(由 api 进程内嵌)")
	}

	a, err := app.New(cfg)
	if err != nil {
		log.Fatalf("初始化应用失败: %v", err)
	}
	defer a.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	log.Printf("worker 已启动, 队列=%s topic=%s 并发=%d", cfg.Queue.Type, cfg.Queue.Topic, cfg.Queue.Concurrency)
	if err := a.StartConsumer(ctx); err != nil {
		log.Fatalf("消费者异常退出: %v", err)
	}
	log.Println("worker 已退出")
}
