//go:build !debug

package api

import (
	"axmipusher/internal/app"

	"github.com/gin-gonic/gin"
)

// registerDebugRoutes 默认构建空实现: 调试路由(模拟支付等)仅在 debug 构建注册。
func registerDebugRoutes(_ *gin.Engine, _ *app.App) {}
