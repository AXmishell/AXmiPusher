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
		admin, err := a.Auth.AdminLogin(req.Email, req.Password)
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
