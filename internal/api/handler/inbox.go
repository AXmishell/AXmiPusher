package handler

import (
	"strconv"
	"time"

	"messagepusher/internal/app"
	"messagepusher/internal/api/middleware"
	"messagepusher/internal/models"
	"messagepusher/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// ListInbox 收件箱列表(当前用户, AntD 分页)。
func ListInbox(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.CurrentUser(c)
		if user == nil {
			response.Unauthorized(c, "未登录")
			return
		}
		page, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
		size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		read := c.Query("read") // true/false 过滤
		if page < 1 {
			page = 1
		}
		if size < 1 || size > 100 {
			size = 20
		}
		q := a.DB.Model(&models.InappMessage{}).Where("tenant_id = ? AND user_id = ?", user.TenantID, user.ID)
		if read == "true" {
			q = q.Where("is_read = ?", true)
		} else if read == "false" {
			q = q.Where("is_read = ?", false)
		}
		var total int64
		q.Count(&total)
		var list []models.InappMessage
		if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error; err != nil {
			response.ServerError(c, "查询失败")
			return
		}
		response.OK(c, gin.H{"data": list, "total": total, "success": true})
	}
}

// UnreadInboxCount 未读站内信数量。
func UnreadInboxCount(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.CurrentUser(c)
		if user == nil {
			response.Unauthorized(c, "未登录")
			return
		}
		var count int64
		a.DB.Model(&models.InappMessage{}).
			Where("tenant_id = ? AND user_id = ? AND is_read = ?", user.TenantID, user.ID, false).
			Count(&count)
		response.OK(c, gin.H{"unread": count})
	}
}

// MarkInboxRead 标记已读。
func MarkInboxRead(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.CurrentUser(c)
		if user == nil {
			response.Unauthorized(c, "未登录")
			return
		}
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的 ID")
			return
		}
		now := time.Now()
		res := a.DB.Model(&models.InappMessage{}).
			Where("id = ? AND tenant_id = ? AND user_id = ?", id, user.TenantID, user.ID).
			Updates(map[string]any{"is_read": true, "read_at": &now})
		if res.RowsAffected == 0 {
			response.NotFound(c, "消息不存在")
			return
		}
		response.OK(c, gin.H{"ok": true})
	}
}

// MarkAllInboxRead 全部标记已读。
func MarkAllInboxRead(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.CurrentUser(c)
		if user == nil {
			response.Unauthorized(c, "未登录")
			return
		}
		now := time.Now()
		a.DB.Model(&models.InappMessage{}).
			Where("tenant_id = ? AND user_id = ? AND is_read = ?", user.TenantID, user.ID, false).
			Updates(map[string]any{"is_read": true, "read_at": &now})
		response.OK(c, gin.H{"ok": true})
	}
}
