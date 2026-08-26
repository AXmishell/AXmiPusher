//go:build debug

package handler

import (
	"strconv"
	"time"

	"axmipusher/internal/app"
	"axmipusher/internal/pkg/response"
	"axmipusher/internal/service"

	"github.com/gin-gonic/gin"
)

// SimulatePay 开发调试: 模拟易支付回调(仅 debug 构建启用)。
// 生产构建(不带 debug 标签)不编译本文件, 对应路由由 router_debug.go 注册,
// 默认构建通过 router_nodebug.go 空桩保证路由装配两端一致编译。
func SimulatePay(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的订单 ID")
			return
		}
		order, err := a.Pay.QueryOrder(resolveTenantID(c), id)
		if err != nil {
			response.NotFound(c, "订单不存在")
			return
		}
		cfg, err := a.Pay.LoadConfig()
		if err != nil {
			response.BadRequest(c, "易支付未配置")
			return
		}
		params := map[string]string{
			"pid":          cfg.PID,
			"trade_no":     "MOCK" + strconv.FormatInt(time.Now().UnixMilli(), 10),
			"out_trade_no": order.OutTradeNo,
			"type":         order.Type,
			"name":         "模拟支付",
			"money":        strconv.FormatFloat(order.Amount, 'f', 2, 64),
			"trade_status": "TRADE_SUCCESS",
		}
		params["sign"] = service.SignParams(params, cfg.Key)
		if err := a.Pay.HandleNotify(params); err != nil {
			response.ServerError(c, "模拟回调失败: "+err.Error())
			return
		}
		response.OK(c, gin.H{"ok": true, "out_trade_no": order.OutTradeNo})
	}
}
