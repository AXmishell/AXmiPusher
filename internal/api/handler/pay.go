package handler

import (
	"errors"
	"net/http"
	"strconv"

	"messagepusher/internal/app"
	"messagepusher/internal/models"
	"messagepusher/internal/pkg/response"
	"messagepusher/internal/service"

	"github.com/gin-gonic/gin"
)

// ListPayPlans 可用套餐列表(公开)。
func ListPayPlans(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		plans, err := a.Pay.ListPlans()
		if err != nil {
			response.ServerError(c, "查询失败")
			return
		}
		response.OK(c, gin.H{"data": plans, "total": len(plans), "success": true})
	}
}

// CreatePayOrder 创建支付订单。
func CreatePayOrder(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			PlanID uint64 `json:"plan_id" binding:"required"`
			Type   string `json:"type"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		order, payURL, err := a.Pay.CreateOrder(resolveTenantID(c), req.PlanID, req.Type)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrEpayNotConfigured):
				response.BadRequest(c, "易支付尚未配置")
			case errors.Is(err, service.ErrPlanInvalid):
				response.BadRequest(c, "套餐不存在或已下架")
			default:
				response.ServerError(c, "创建订单失败: "+err.Error())
			}
			return
		}
		Audit(a.DB, c, currentUserID(c), currentUserEmail(c), "pay.order_create",
			gin.H{"order_id": order.ID, "out_trade_no": order.OutTradeNo, "amount": order.Amount})
		response.OK(c, gin.H{"order": order, "pay_url": payURL})
	}
}

// QueryPayOrder 查询订单状态(前端轮询)。
func QueryPayOrder(a *app.App) gin.HandlerFunc {
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
		response.OK(c, order)
	}
}

// GetSubscription 当前订阅。
func GetSubscription(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		sub, plan, err := a.Pay.CurrentSubscription(resolveTenantID(c))
		if err != nil {
			response.ServerError(c, "查询失败")
			return
		}
		if sub == nil {
			response.OK(c, gin.H{"subscription": nil, "plan": nil})
			return
		}
		response.OK(c, gin.H{"subscription": sub, "plan": plan})
	}
}

// PayNotify 易支付回调(公开, 服务端验签)。
// 上游约定: 返回文本 "success" 表示停止重试。
func PayNotify(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 注意: r.PostForm 字段需要先 ParseForm 才会填充。
		if err := c.Request.ParseForm(); err != nil {
			c.String(http.StatusOK, "fail")
			return
		}
		params := map[string]string{}
		for k, v := range c.Request.PostForm {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
		if err := a.Pay.HandleNotify(params); err != nil {
			// 业务失败返回非 success, 上游会重试。
			c.String(http.StatusOK, "fail")
			return
		}
		Audit(a.DB, c, 0, "system:epay-notify", "pay.notify_success",
			gin.H{"out_trade_no": params["out_trade_no"]})
		c.String(http.StatusOK, "success")
	}
}

// PayReturn 易支付回跳(浏览器)。
func PayReturn(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		outTradeNo := c.Query("out_trade_no")
		var order models.PaymentOrder
		if outTradeNo != "" {
			a.DB.Where("out_trade_no = ?", outTradeNo).First(&order)
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		if order.Status == "paid" {
			c.String(http.StatusOK, "<html><head><meta charset='utf-8'></head><body style='font-family:sans-serif;text-align:center;padding-top:80px'><h2>✅ 支付成功</h2><p>订单 %s 已到账, 套餐已生效。</p><p><a href='/'>返回平台</a></p></body></html>", order.OutTradeNo)
			return
		}
		c.String(http.StatusOK, "<html><head><meta charset='utf-8'></head><body style='font-family:sans-serif;text-align:center;padding-top:80px'><h2>支付结果确认中</h2><p>订单 %s 状态尚未确认, 请稍后刷新或联系客服。</p><p><a href='/'>返回平台</a></p></body></html>", outTradeNo)
	}
}

