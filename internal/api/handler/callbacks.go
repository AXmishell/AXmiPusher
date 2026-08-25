package handler

import (
	"strconv"

	"messagepusher/internal/app"
	"messagepusher/internal/models"
	"messagepusher/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// callbackRequest 注册回调请求。
type callbackRequest struct {
	URL    string   `json:"url" binding:"required,url"`
	Secret string   `json:"secret"`
	Events []string `json:"events"`
}

// ListCallbacks 列出回调订阅。
func ListCallbacks(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var subs []models.WebhookSubscription
		if err := a.DB.Where("tenant_id = ?", resolveTenantID(c)).Order("id DESC").Find(&subs).Error; err != nil {
			response.ServerError(c, "查询失败")
			return
		}
		response.OK(c, gin.H{"data": subs, "total": len(subs), "success": true})
	}
}

// CreateCallback 注册回调订阅。
func CreateCallback(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req callbackRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		events := req.Events
		if len(events) == 0 {
			events = []string{"success", "failed"}
		}
		sub := models.WebhookSubscription{
			TenantID: resolveTenantID(c),
			URL:      req.URL,
			Secret:   req.Secret,
			Events:   joinStrings(events),
			Status:   models.StatusActive,
		}
		if err := a.DB.Create(&sub).Error; err != nil {
			response.ServerError(c, "创建失败")
			return
		}
		response.OK(c, sub)
	}
}

// DeleteCallback 删除回调订阅。
func DeleteCallback(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的 ID")
			return
		}
		if err := a.DB.Where("id = ? AND tenant_id = ?", id, resolveTenantID(c)).Delete(&models.WebhookSubscription{}).Error; err != nil {
			response.ServerError(c, "删除失败")
			return
		}
		response.OK(c, gin.H{"ok": true})
	}
}

func joinStrings(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}
