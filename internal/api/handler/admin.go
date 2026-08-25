package handler

import (
	"strconv"
	"time"

	"messagepusher/internal/app"
	"messagepusher/internal/models"
	"messagepusher/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// ListTenants 租户列表(含消息量统计)。
func ListTenants(a *app.App) gin.HandlerFunc {
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
		a.DB.Model(&models.Tenant{}).Count(&total)
		var tenants []models.Tenant
		if err := a.DB.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&tenants).Error; err != nil {
			response.ServerError(c, "查询失败")
			return
		}
		// 补充每个租户的用户数与消息量。
		type row struct {
			TenantID uint64
			Cnt      int64
		}
		ids := make([]uint64, 0, len(tenants))
		for _, t := range tenants {
			ids = append(ids, t.ID)
		}
		userCnt := map[uint64]int64{}
		msgCnt := map[uint64]int64{}
		if len(ids) > 0 {
			var urows []row
			a.DB.Model(&models.User{}).Select("tenant_id, COUNT(*) as cnt").Where("tenant_id IN ?", ids).Group("tenant_id").Scan(&urows)
			for _, r := range urows {
				userCnt[r.TenantID] = r.Cnt
			}
			// 消息量从 MessageStore 统计(本地模式: SQLite 同库)。
			since := time.Now().Add(-24 * time.Hour)
			for _, t := range tenants {
				if stats, err := a.Store.StatsByStatus(c.Request.Context(), t.ID, since, time.Now()); err == nil {
					for _, v := range stats {
						msgCnt[t.ID] += v
					}
				}
			}
		}
		list := make([]gin.H, 0, len(tenants))
		for _, t := range tenants {
			list = append(list, gin.H{
				"id": t.ID, "name": t.Name, "status": t.Status,
				"plan_id": t.PlanID, "created_at": t.CreatedAt,
				"user_count": userCnt[t.ID], "msg_24h": msgCnt[t.ID],
			})
		}
		response.OK(c, gin.H{"data": list, "total": total, "success": true})
	}
}

// SetTenantStatus 启用/禁用租户。
func SetTenantStatus(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的租户 ID")
			return
		}
		var req struct {
			Status string `json:"status" binding:"required,oneof=active disabled"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "status 必须是 active 或 disabled")
			return
		}
		res := a.DB.Model(&models.Tenant{}).Where("id = ?", id).Update("status", req.Status)
		if res.RowsAffected == 0 {
			response.NotFound(c, "租户不存在")
			return
		}
		Audit(a.DB, c, currentUserID(c), currentUserEmail(c), "tenant.set_status",
			gin.H{"tenant_id": id, "status": req.Status})
		response.OK(c, gin.H{"ok": true})
	}
}

// ListUsers 用户列表(可按租户过滤)。
func ListUsers(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
		size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		tenantID, _ := strconv.ParseUint(c.Query("tenant_id"), 10, 64)
		if page < 1 {
			page = 1
		}
		if size < 1 || size > 100 {
			size = 20
		}
		q := a.DB.Model(&models.User{})
		if tenantID > 0 {
			q = q.Where("tenant_id = ?", tenantID)
		}
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
		Audit(a.DB, c, currentUserID(c), currentUserEmail(c), "user.set_status",
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
		// 全租户消息量: 遍历租户统计(本地模式)。
		var tenants []models.Tenant
		a.DB.Find(&tenants)
		total, success, failed := int64(0), int64(0), int64(0)
		for _, t := range tenants {
			if stats, err := a.Store.StatsByStatus(c.Request.Context(), t.ID, since, now); err == nil {
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
		var tenantCount, userCount, templateCount int64
		a.DB.Model(&models.Tenant{}).Count(&tenantCount)
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
			"tenants":         tenantCount,
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
