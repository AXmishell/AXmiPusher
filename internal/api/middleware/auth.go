// Package middleware 提供认证、租户隔离、限流、安装状态等中间件。
package middleware

import (
	"strings"

	"axmipusher/internal/models"
	"axmipusher/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ContextKey 上下文键。
type ContextKey string

// 上下文键常量。
const (
	CtxUser   ContextKey = "user"
	CtxTenant ContextKey = "tenant"
	CtxAPIKey ContextKey = "api_key"
	CtxAdmin  ContextKey = "admin"
)

// AuthService 认证服务接口(由 service 层实现)。
type AuthService interface {
	// ParseToken 解析 JWT 返回用户。
	ParseToken(token string) (*models.User, error)
	// ResolveAPIKey 解析 API Key 返回 key(tenant_id 即归属用户 ID)。
	ResolveAPIKey(key string) (*models.APIKey, error)
}

// AdminAuthService 管理员认证服务接口(由 service 层实现)。
type AdminAuthService interface {
	// ParseAdminToken 解析管理员 JWT 返回管理员。
	ParseAdminToken(token string) (*models.Admin, error)
}

// RequireAuth 要求 JWT 登录态。
// getAuth 为惰性取值函数(应用容器重建后返回最新 AuthService)。
func RequireAuth(getAuth func() AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c)
		if token == "" {
			response.Unauthorized(c, "缺少认证凭证")
			c.Abort()
			return
		}
		user, err := getAuth().ParseToken(token)
		if err != nil {
			response.Unauthorized(c, "凭证无效或已过期")
			c.Abort()
			return
		}
		c.Set(string(CtxUser), user)
		c.Next()
	}
}

// RequireAPIKey 要求 API Key(服务端调用)。
func RequireAPIKey(getAuth func() AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := extractBearer(c)
		if key == "" {
			response.Unauthorized(c, "缺少 API Key")
			c.Abort()
			return
		}
		apiKey, err := getAuth().ResolveAPIKey(key)
		if err != nil || apiKey == nil {
			response.Unauthorized(c, "API Key 无效")
			c.Abort()
			return
		}
		c.Set(string(CtxAPIKey), apiKey)
		c.Set(string(CtxTenant), apiKey.TenantID)
		c.Next()
	}
}

// RequireAuthOrAPIKey JWT 登录态 或 API Key 均可(网页端与服务端通用)。
func RequireAuthOrAPIKey(getAuth func() AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c)
		if token == "" {
			response.Unauthorized(c, "缺少认证凭证")
			c.Abort()
			return
		}
		auth := getAuth()
		// 优先尝试 API Key(服务端场景)。
		if apiKey, err := auth.ResolveAPIKey(token); err == nil && apiKey != nil {
			c.Set(string(CtxAPIKey), apiKey)
			c.Set(string(CtxTenant), apiKey.TenantID)
			c.Next()
			return
		}
		// 再尝试 JWT(网页场景)。
		user, err := auth.ParseToken(token)
		if err != nil {
			response.Unauthorized(c, "凭证无效或已过期")
			c.Abort()
			return
		}
		c.Set(string(CtxUser), user)
		c.Next()
	}
}

// RequireRole 要求指定角色(仅用于 JWT 登录态)。
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := c.Get(string(CtxUser))
		if !ok {
			response.Unauthorized(c, "未登录")
			c.Abort()
			return
		}
		user := u.(*models.User)
		for _, r := range roles {
			if user.Role == r {
				c.Next()
				return
			}
		}
		response.Forbidden(c, "无权限执行此操作")
		c.Abort()
	}
}

// RequireAdminAuth 要求管理员 JWT 登录态。
// getAuth 为惰性取值函数(应用容器重建后返回最新 AuthService)。
func RequireAdminAuth(getAuth func() AdminAuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c)
		if token == "" {
			response.Unauthorized(c, "缺少认证凭证")
			c.Abort()
			return
		}
		admin, err := getAuth().ParseAdminToken(token)
		if err != nil {
			response.Unauthorized(c, "凭证无效或已过期")
			c.Abort()
			return
		}
		c.Set(string(CtxAdmin), admin)
		c.Next()
	}
}

// RequireAdminRole 要求指定管理员角色(仅用于管理员 JWT 登录态)。
func RequireAdminRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		a, ok := c.Get(string(CtxAdmin))
		if !ok {
			response.Unauthorized(c, "未登录")
			c.Abort()
			return
		}
		admin := a.(*models.Admin)
		for _, r := range roles {
			if admin.Role == r {
				c.Next()
				return
			}
		}
		response.Forbidden(c, "无权限执行此操作")
		c.Abort()
	}
}

// RequireInstalled 未安装时拒绝业务 API。
func RequireInstalled() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !installed() {
			response.Fail(c, 503, 50300, "平台尚未安装, 请先访问 /install 完成安装")
			c.Abort()
			return
		}
		c.Next()
	}
}

// installed 由 install 包注入的状态判断。
var installed = func() bool { return true }

// SetInstalledChecker 注入安装状态判断函数。
func SetInstalledChecker(fn func() bool) {
	installed = fn
}

func extractBearer(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

// DB 从上下文获取 gorm.DB 的辅助函数(供 handler 使用)。
func DB(c *gin.Context) *gorm.DB {
	if v, ok := c.Get("_db"); ok {
		if db, ok := v.(*gorm.DB); ok {
			return db
		}
	}
	return nil
}

// CurrentUser 从上下文取当前用户。
func CurrentUser(c *gin.Context) *models.User {
	if v, ok := c.Get(string(CtxUser)); ok {
		if u, ok := v.(*models.User); ok {
			return u
		}
	}
	return nil
}

// CurrentAdmin 从上下文取当前管理员。
func CurrentAdmin(c *gin.Context) *models.Admin {
	if v, ok := c.Get(string(CtxAdmin)); ok {
		if a, ok := v.(*models.Admin); ok {
			return a
		}
	}
	return nil
}

// CurrentTenantID 从上下文取当前归属用户 ID(API Key 场景写入, JWT 场景用 CurrentUser.ID)。
func CurrentTenantID(c *gin.Context) uint64 {
	if v, ok := c.Get(string(CtxTenant)); ok {
		if id, ok := v.(uint64); ok {
			return id
		}
	}
	return 0
}
