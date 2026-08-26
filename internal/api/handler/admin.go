package handler

import (
	"strconv"
	"time"

	"messagepusher/internal/app"
	"messagepusher/internal/models"
	"messagepusher/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// ListUsers 用户列表。
func ListUsers(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
		size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		if page < 1 {
			page = 1
		}
		if size < 1 || size > 100 {
			size = 20
		}
		q := a.DB.Model(&models.User{})
		var total int64
		q.Count(&total)
		var users []models.User
		if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&users).Error; err != nil {
			response.ServerError(c, "查询失败")
			return
		}
		response.OK(c, gin.H{"data": users, "total": total, "success": true})
	}
}

// SetUserStatus 启用/禁用用户。
func SetUserStatus(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的用户 ID")
			return
		}
		var req struct {
			Status string `json:"status" binding:"required,oneof=active disabled"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "status 必须是 active 或 disabled")
			return
		}
		var user models.User
		if err := a.DB.First(&user, id).Error; err != nil {
			response.NotFound(c, "用户不存在")
			return
		}
		if user.Role == models.RolePlatformAdmin {
			response.Forbidden(c, "不能禁用平台管理员")
			return
		}
		if err := a.DB.Model(&user).Update("status", req.Status).Error; err != nil {
			response.ServerError(c, "操作失败")
			return
		}
		Audit(a.DB, c, currentAdminID(c), currentAdminEmail(c), "user.set_status",
			gin.H{"user_id": id, "status": req.Status})
		response.OK(c, gin.H{"ok": true})
	}
}

// AdminStats 平台全局统计。
func AdminStats(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now()
		since := now.Add(-24 * time.Hour)
		if s := c.Query("since"); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				since = t
			}
		}
		// 平台全量消息量(逐用户统计汇总)。
		var users []models.User
		a.DB.Find(&users)
		total, success, failed := int64(0), int64(0), int64(0)
		for _, u := range users {
			if stats, err := a.Store.StatsByStatus(c.Request.Context(), u.ID, since, now); err == nil {
				for k, v := range stats {
					total += v
					if k == "SUCCESS" {
						success += v
					}
					if k == "FAILED" || k == "DEAD" {
						failed += v
					}
				}
			}
		}
		var userCount, templateCount int64
		a.DB.Model(&models.User{}).Count(&userCount)
		a.DB.Model(&models.Template{}).Count(&templateCount)
		var pendingReviews int64
		a.DB.Model(&models.TemplateVersion{}).Where("review_status = ?", models.StatusPending).Count(&pendingReviews)
		rate := 0.0
		if total > 0 {
			rate = float64(success) / float64(total) * 100
		}
		response.OK(c, gin.H{
			"messages":        gin.H{"total": total, "success": success, "failed": failed, "success_rate": round1(rate)},
			"users":           userCount,
			"templates":       templateCount,
			"pending_reviews": pendingReviews,
		})
	}
}

// ListAuditLogs 审计日志列表。
func ListAuditLogs(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
		size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		if page < 1 {
			page = 1
		}
		if size < 1 || size > 100 {
			size = 20
		}
		var total int64
		a.DB.Model(&models.AuditLog{}).Count(&total)
		var logs []models.AuditLog
		if err := a.DB.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&logs).Error; err != nil {
			response.ServerError(c, "查询失败")
			return
		}
		response.OK(c, gin.H{"data": logs, "total": total, "success": true})
	}
}
