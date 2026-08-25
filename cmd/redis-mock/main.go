// 本地调试工具: 在固定端口启动 miniredis(模拟单机 Redis)。
package main

import (
	"fmt"
	"log"

	"github.com/alicebob/miniredis/v2"
)

func main() {
	mr := miniredis.NewMiniRedis()
	if err := mr.StartAddr("127.0.0.1:16379"); err != nil {
		log.Fatalf("启动 miniredis 失败: %v", err)
	}
	fmt.Println("mock redis 监听 127.0.0.1:16379")
	select {}
}
