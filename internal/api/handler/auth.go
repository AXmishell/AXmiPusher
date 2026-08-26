// Package handler HTTP 处理器层。
package handler

import (
	"errors"

	"messagepusher/internal/app"
	"messagepusher/internal/api/middleware"
	"messagepusher/internal/models"
	"messagepusher/internal/pkg/response"
	"messagepusher/internal/service"

	"github.com/gin-gonic/gin"
)

// registerRequest 注册请求。
type registerRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required"`
	TenantName string `json:"tenant_name"`
	Nickname   string `json:"nickname"`
}

// Register 开放注册。
func Register(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req registerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		user, err := a.Auth.Register(req.Email, req.Password, req.TenantName, req.Nickname)
		if err != nil {
			response.Conflict(c, err.Error())
			return
		}
		token, err := a.Auth.CreateToken(user)
		if err != nil {
			response.ServerError(c, "签发令牌失败")
			return
		}
		response.OK(c, gin.H{
			"token": token,
			"user":  user,
		})
	}
}

// loginRequest 登录请求。
type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 登录。
func Login(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误")
			return
		}
		user, err := a.Auth.Login(req.Email, req.Password, c.ClientIP())
		if err != nil {
			response.Unauthorized(c, "邮箱或密码错误")
			return
		}
		token, err := a.Auth.CreateToken(user)
		if err != nil {
			response.ServerError(c, "签发令牌失败")
			return
		}
		response.OK(c, gin.H{"token": token, "user": user})
	}
}

// changePasswordRequest 修改密码请求。
type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword 修改当前用户密码(登录态)。
func ChangePassword(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.CurrentUser(c)
		if user == nil {
			response.Unauthorized(c, "未登录")
			return
		}
		var req changePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		if err := a.Auth.ChangePassword(user.ID, req.OldPassword, req.NewPassword); err != nil {
			if errors.Is(err, service.ErrBadOldPass) {
				response.BadRequest(c, "旧密码不正确")
			} else {
				response.BadRequest(c, err.Error())
			}
			return
		}
		Audit(a.DB, c, user.ID, user.Email, "auth.change_password", gin.H{"user_id": user.ID})
		response.OK(c, gin.H{"ok": true})
	}
}

// Me 当前登录用户信息。
func Me(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.CurrentUser(c)
		if user == nil {
			response.Unauthorized(c, "未登录")
			return
		}
		response.OK(c, gin.H{
			"user":     user,
			"is_admin": user.Role == models.RolePlatformAdmin,
		})
	}
}

// updateProfileRequest 更新账户资料请求。
type updateProfileRequest struct {
	Name     string `json:"name"`     // 用户名(原租户名语义)
	Nickname string `json:"nickname"` // 昵称
	QQ       string `json:"qq"`       // QQ 号码(可空)
}

// UpdateProfile 更新用户名/昵称/QQ(登录态)。
func UpdateProfile(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.CurrentUser(c)
		if user == nil {
			response.Unauthorized(c, "未登录")
			return
		}
		var req updateProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		if req.Name != "" && len(req.Name) > 128 {
			response.BadRequest(c, "用户名过长(最多 128 字)")
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
		updated, err := a.Auth.UpdateProfile(user.ID, req.Name, req.Nickname, req.QQ)
		if err != nil {
			response.ServerError(c, "更新失败: "+err.Error())
			return
		}
		Audit(a.DB, c, user.ID, user.Email, "user.update_profile", gin.H{"user_id": user.ID})
		response.OK(c, gin.H{"user": updated})
	}
}

// changeEmailRequest 修改邮箱请求。
type changeEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ChangeEmail 修改登录邮箱(登录态, 唯一性校验)。
func ChangeEmail(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.CurrentUser(c)
		if user == nil {
			response.Unauthorized(c, "未登录")
			return
		}
		var req changeEmailRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		updated, err := a.Auth.ChangeEmail(user.ID, req.Email)
		if err != nil {
			if errors.Is(err, service.ErrEmailExists) {
				response.Conflict(c, "邮箱已被注册")
				return
			}
			response.ServerError(c, "修改失败: "+err.Error())
			return
		}
		Audit(a.DB, c, user.ID, updated.Email, "user.change_email", gin.H{"user_id": user.ID, "old": user.Email, "new": updated.Email})
		response.OK(c, gin.H{"user": updated})
	}
}
