package handler

import (
	"strconv"
	"strings"
	"time"

	"axmipusher/internal/app"
	"axmipusher/internal/models"
	"axmipusher/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
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

// CreateUser 新增用户(管理后台创建, 默认角色 tenant_user, 状态 active)。
func CreateUser(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required"`
			Nickname string `json:"nickname"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		req.Email = strings.TrimSpace(req.Email)
		if !strings.Contains(req.Email, "@") {
			response.BadRequest(c, "邮箱格式不正确")
			return
		}
		if len(req.Password) < 8 {
			response.BadRequest(c, "密码长度至少 8 位")
			return
		}
		nickname := strings.TrimSpace(req.Nickname)
		if nickname == "" {
			nickname = req.Email // 与注册逻辑一致: 用户名默认邮箱
		}
		var cnt int64
		a.DB.Model(&models.User{}).Where("email = ?", req.Email).Count(&cnt)
		if cnt > 0 {
			response.Conflict(c, "邮箱已被占用")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
		if err != nil {
			response.ServerError(c, "密码加密失败")
			return
		}
		user := models.User{
			Email:        req.Email,
			PasswordHash: string(hash),
			Nickname:     nickname,
			Role:         models.RoleTenantUser,
			Status:       models.StatusActive,
		}
		if err := a.DB.Create(&user).Error; err != nil {
			response.Conflict(c, "创建失败: "+err.Error())
			return
		}
		Audit(a.DB, c, currentAdminID(c), currentAdminEmail(c), "user.create",
			gin.H{"user_id": user.ID, "email": req.Email})
		response.OK(c, gin.H{"id": user.ID, "email": user.Email})
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

// UpdateUser 编辑用户资料(邮箱/用户名)与重置密码。
// 字段留空表示不修改; password 非空则重置为 bcrypt 新哈希(用户下次登录用新密码)。
func UpdateUser(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的用户 ID")
			return
		}
		var req struct {
			Email    string `json:"email"`
			Nickname string `json:"nickname"`
			Password string `json:"password"` // 非空则重置密码
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		var user models.User
		if err := a.DB.First(&user, id).Error; err != nil {
			response.NotFound(c, "用户不存在")
			return
		}
		if user.Role == models.RolePlatformAdmin {
			response.Forbidden(c, "不能编辑平台管理员")
			return
		}
		updates := map[string]interface{}{}
		// 邮箱: 格式 + 唯一性(排除自身)。
		if req.Email != "" && req.Email != user.Email {
			if !strings.Contains(req.Email, "@") {
				response.BadRequest(c, "邮箱格式不正确")
				return
			}
			var cnt int64
			a.DB.Model(&models.User{}).Where("email = ? AND id != ?", req.Email, id).Count(&cnt)
			if cnt > 0 {
				response.Conflict(c, "邮箱已被占用")
				return
			}
			updates["email"] = req.Email
		}
		// 用户名。
		if req.Nickname != "" && req.Nickname != user.Nickname {
			updates["nickname"] = req.Nickname
		}
		// 重置密码。
		if req.Password != "" {
			if len(req.Password) < 8 {
				response.BadRequest(c, "新密码长度至少 8 位")
				return
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
			if err != nil {
				response.ServerError(c, "密码加密失败")
				return
			}
			updates["password_hash"] = string(hash)
		}
		if len(updates) == 0 {
			response.BadRequest(c, "没有需要更新的字段")
			return
		}
		if err := a.DB.Model(&user).Updates(updates).Error; err != nil {
			response.ServerError(c, "保存失败: "+err.Error())
			return
		}
		Audit(a.DB, c, currentAdminID(c), currentAdminEmail(c), "user.update",
			gin.H{"user_id": id, "email_changed": req.Email != "", "nickname_changed": req.Nickname != "", "password_reset": req.Password != ""})
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
