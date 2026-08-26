package handler

import (
	"errors"

	"axmipusher/internal/api/middleware"
	"axmipusher/internal/app"
	"axmipusher/internal/pkg/response"
	"axmipusher/internal/service"

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
		// 已启用两步验证: 返回临时凭证, 前端进入验证码步骤。
		if admin.TotpEnabled {
			pending, err := a.Auth.CreateTotpPendingToken(admin.ID, true)
			if err != nil {
				response.ServerError(c, "签发验证凭证失败")
				return
			}
			response.OK(c, gin.H{"need_totp": true, "totp_token": pending})
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

// AdminLoginTotp 管理员登录第二步: 校验 TOTP 验证码后签发正式 token。
func AdminLoginTotp(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req totpLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误")
			return
		}
		id, isAdmin, err := a.Auth.ParseTotpPendingToken(req.TotpToken)
		if err != nil || !isAdmin {
			response.Unauthorized(c, "验证凭证无效或已过期")
			return
		}
		admin, err := a.Auth.AdminLoginTotp(id, req.Code, c.ClientIP())
		if err != nil {
			response.Unauthorized(c, err.Error())
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

// AdminSetupTotp 生成管理员 TOTP 密钥与二维码(登录态, 仅未启用时)。
func AdminSetupTotp(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.CurrentAdmin(c)
		if admin == nil {
			response.Unauthorized(c, "未登录")
			return
		}
		secret, otpauthURL, qrDataURL, err := a.Auth.SetupTotp(admin.ID, true)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		response.OK(c, gin.H{"secret": secret, "otpauth_url": otpauthURL, "qr_data_url": qrDataURL})
	}
}

// AdminConfirmTotp 确认启用管理员两步验证。
func AdminConfirmTotp(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.CurrentAdmin(c)
		if admin == nil {
			response.Unauthorized(c, "未登录")
			return
		}
		var req totpSetupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误")
			return
		}
		if err := a.Auth.ConfirmTotp(admin.ID, true, req.Code); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		response.OK(c, gin.H{"totp_enabled": true})
	}
}

// AdminDisableTotp 关闭管理员两步验证。
func AdminDisableTotp(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.CurrentAdmin(c)
		if admin == nil {
			response.Unauthorized(c, "未登录")
			return
		}
		var req totpSetupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误")
			return
		}
		if err := a.Auth.DisableTotp(admin.ID, true, req.Code); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		response.OK(c, gin.H{"totp_enabled": false})
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
