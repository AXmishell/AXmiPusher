package handler

import (
	"strconv"

	"messagepusher/internal/api/middleware"
	"messagepusher/internal/app"
	"messagepusher/internal/pkg/response"
	"messagepusher/internal/service"

	"github.com/gin-gonic/gin"
)

// ListAdmins 管理员列表(仅超管, AntD Pro 分页)。
func ListAdmins(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
		size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		if page < 1 {
			page = 1
		}
		if size < 1 || size > 100 {
			size = 20
		}
		admins, err := a.Auth.ListAdmins()
		if err != nil {
			response.ServerError(c, "查询失败")
			return
		}
		// 分页切片。
		start := (page - 1) * size
		if start > len(admins) {
			start = len(admins)
		}
		end := start + size
		if end > len(admins) {
			end = len(admins)
		}
		response.OK(c, gin.H{"data": admins[start:end], "total": len(admins), "success": true})
	}
}

// createAdminRequest 创建管理员请求。
type createAdminRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname"`
}

// CreateAdmin 创建管理员(仅超管)。
func CreateAdmin(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createAdminRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		admin, err := a.Auth.CreateAdmin(req.Email, req.Password, req.Nickname)
		if err != nil {
			if err == service.ErrEmailExists {
				response.Conflict(c, err.Error())
			} else {
				response.BadRequest(c, "创建失败: "+err.Error())
			}
			return
		}
		actor := middleware.CurrentAdmin(c)
		Audit(a.DB, c, actor.ID, actor.Email, "admin.create", gin.H{"admin_id": admin.ID, "email": admin.Email})
		response.OK(c, admin)
	}
}

// setAdminStatusRequest 设置管理员状态请求。
type setAdminStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active disabled"`
}

// SetAdminStatus 启用/禁用管理员(仅超管)。
func SetAdminStatus(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor := middleware.CurrentAdmin(c)
		if actor == nil {
			response.Unauthorized(c, "未登录")
			return
		}
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的管理员 ID")
			return
		}
		var req setAdminStatusRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "status 必须是 active 或 disabled")
			return
		}
		if err := a.Auth.SetAdminStatus(actor.ID, id, req.Status); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		Audit(a.DB, c, actor.ID, actor.Email, "admin.set_status", gin.H{"admin_id": id, "status": req.Status})
		response.OK(c, gin.H{"ok": true})
	}
}

// resetAdminPasswordRequest 重置管理员密码请求。
type resetAdminPasswordRequest struct {
	Password string `json:"password" binding:"required"`
}

// ResetAdminPassword 重置管理员密码(仅超管, 免旧密码)。
func ResetAdminPassword(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor := middleware.CurrentAdmin(c)
		if actor == nil {
			response.Unauthorized(c, "未登录")
			return
		}
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的管理员 ID")
			return
		}
		var req resetAdminPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		if err := a.Auth.ResetAdminPassword(id, req.Password); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		Audit(a.DB, c, actor.ID, actor.Email, "admin.reset_password", gin.H{"admin_id": id})
		response.OK(c, gin.H{"ok": true})
	}
}
