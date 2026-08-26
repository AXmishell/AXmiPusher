//go:build debug

package api

import (
	"messagepusher/internal/api/handler"
	"messagepusher/internal/api/middleware"
	"messagepusher/internal/app"

	"github.com/gin-gonic/gin"
)

// registerDebugRoutes 注册仅 debug 构建可用的调试路由。
// 默认构建由 router_nodebug.go 提供空实现, 保证两端路由装配一致编译。
func registerDebugRoutes(r *gin.Engine, a *app.App) {
	getAuth := func() middleware.AuthService { return a.Auth }
	biz := r.Group("/api/v1", middleware.RequireInstalled())
	pay := biz.Group("/pay", middleware.RequireAuth(getAuth))
	pay.POST("/orders/:id/simulate", handler.SimulatePay(a)) // 本地调试用
}
