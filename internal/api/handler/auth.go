// Package handler HTTP 处理器层。
package handler

import (
	"messagepusher/internal/app"
	"messagepusher/internal/api/middleware"
	"messagepusher/internal/models"
	"messagepusher/internal/pkg/response"

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
		user, tenant, err := a.Auth.Register(req.Email, req.Password, req.TenantName, req.Nickname)
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
			"token":    token,
			"user":     user,
			"tenant":   gin.H{"id": tenant.ID, "name": tenant.Name},
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
		user, err := a.Auth.Login(req.Email, req.Password)
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

// Me 当前登录用户信息。
func Me(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.CurrentUser(c)
		if user == nil {
			response.Unauthorized(c, "未登录")
			return
		}
		var tenant models.Tenant
		if user.TenantID > 0 {
			a.DB.First(&tenant, user.TenantID)
		}
		response.OK(c, gin.H{
			"user":   user,
			"tenant": tenant,
			"is_admin": user.Role == models.RolePlatformAdmin,
		})
	}
}
