package handler

import (
	"errors"

	"messagepusher/internal/api/middleware"
	"messagepusher/internal/app"
	"messagepusher/internal/pkg/response"
	"messagepusher/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminLogin 管理员登录(独立管理员体系)。
func AdminLogin(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误")
			return
		}
		admin, err := a.Auth.AdminLogin(req.Email, req.Password, c.ClientIP())
		if err != nil {
			response.Unauthorized(c, "邮箱或密码错误")
			return
		}
		token, err := a.Auth.CreateAdminToken(admin)
		if err != nil {
			response.ServerError(c, "签发令牌失败")
			return
		}
		response.OK(c, gin.H{"token": token, "admin": admin})
	}
}

// AdminMe 当前管理员信息。
func AdminMe(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.CurrentAdmin(c)
		if admin == nil {
			response.Unauthorized(c, "未登录")
			return
		}
		response.OK(c, gin.H{"admin": admin})
	}
}

// AdminChangePassword 修改当前管理员密码(登录态)。
func AdminChangePassword(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.CurrentAdmin(c)
		if admin == nil {
			response.Unauthorized(c, "未登录")
			return
		}
		var req changePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		if err := a.Auth.AdminChangePassword(admin.ID, req.OldPassword, req.NewPassword); err != nil {
			if errors.Is(err, service.ErrBadOldPass) {
				response.BadRequest(c, "旧密码不正确")
			} else {
				response.BadRequest(c, err.Error())
			}
			return
		}
		Audit(a.DB, c, admin.ID, admin.Email, "admin.change_password", gin.H{"admin_id": admin.ID})
		response.OK(c, gin.H{"ok": true})
	}
}

// adminUpdateProfileRequest 更新管理员账户资料请求。
type adminUpdateProfileRequest struct {
	Nickname string `json:"nickname"` // 昵称
	QQ       string `json:"qq"`       // QQ 号码(可空)
}

// AdminUpdateProfile 更新当前管理员昵称/QQ(登录态)。
func AdminUpdateProfile(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.CurrentAdmin(c)
		if admin == nil {
			response.Unauthorized(c, "未登录")
			return
		}
		var req adminUpdateProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		if len(req.Nickname) > 64 {
			response.BadRequest(c, "昵称过长(最多 64 字)")
			return
		}
		if len(req.QQ) > 32 {
			response.BadRequest(c, "QQ 号过长")
			return
		}
		updated, err := a.Auth.AdminUpdateProfile(admin.ID, req.Nickname, req.QQ)
		if err != nil {
			response.ServerError(c, "更新失败: "+err.Error())
			return
		}
		Audit(a.DB, c, admin.ID, admin.Email, "admin.update_profile", gin.H{"admin_id": admin.ID})
		response.OK(c, gin.H{"admin": updated})
	}
}

// adminChangeEmailRequest 修改管理员邮箱请求。
type adminChangeEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// AdminChangeEmail 修改当前管理员登录邮箱(admins 表唯一性校验)。
func AdminChangeEmail(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.CurrentAdmin(c)
		if admin == nil {
			response.Unauthorized(c, "未登录")
			return
		}
		var req adminChangeEmailRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		updated, err := a.Auth.AdminChangeEmail(admin.ID, req.Email)
		if err != nil {
			if errors.Is(err, service.ErrEmailExists) {
				response.Conflict(c, "邮箱已被其他管理员使用")
				return
			}
			response.ServerError(c, "修改失败: "+err.Error())
			return
		}
		Audit(a.DB, c, admin.ID, updated.Email, "admin.change_email", gin.H{"admin_id": admin.ID, "old": admin.Email, "new": updated.Email})
		response.OK(c, gin.H{"admin": updated})
	}
}
