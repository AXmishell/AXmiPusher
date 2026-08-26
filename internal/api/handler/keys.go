package handler

import (
	"strconv"
	"time"

	"axmipusher/internal/api/middleware"
	"axmipusher/internal/app"
	"axmipusher/internal/models"
	"axmipusher/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// createKeyRequest 创建 API Key 请求。
type createKeyRequest struct {
	Name      string `json:"name" binding:"required"`
	Scopes    string `json:"scopes"`
	ExpiresIn int    `json:"expires_in"` // 过期天数, 0 为永久
}

// ListKeys 列出当前租户的 API Key。
func ListKeys(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := resolveTenantID(c)
		keys, err := a.Auth.ListAPIKeys(tenantID)
		if err != nil {
			response.ServerError(c, "查询失败")
			return
		}
		response.OK(c, gin.H{"data": keys, "total": len(keys), "success": true})
	}
}

// CreateKey 创建 API Key。
func CreateKey(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createKeyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误")
			return
		}
		var expiresAt *time.Time
		if req.ExpiresIn > 0 {
			t := time.Now().Add(time.Duration(req.ExpiresIn) * 24 * time.Hour)
			expiresAt = &t
		}
		key, plain, err := a.Auth.CreateAPIKey(resolveTenantID(c), req.Name, req.Scopes, expiresAt)
		if err != nil {
			response.ServerError(c, "创建失败")
			return
		}
		// 明文 key 仅此一次返回。
		response.OK(c, gin.H{"key": key, "plain_key": plain})
	}
}

// DeleteKey 吊销 API Key。
func DeleteKey(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的 key ID")
			return
		}
		if err := a.Auth.RevokeAPIKey(resolveTenantID(c), id); err != nil {
			response.ServerError(c, "吊销失败")
			return
		}
		response.OK(c, gin.H{"ok": true})
	}
}

// --- 兼容 key 管理 ---

// createCompatKeyRequest 创建兼容 key。
type createCompatKeyRequest struct {
	Source         string `json:"source" binding:"required"` // serverchan_v1 | serverchan_v2
	ExternalKey    string `json:"external_key"`              // 留空则自动生成
	DefaultChannel string `json:"default_channel"`
	Description    string `json:"description"`
}

// ListCompatKeys 列出兼容 key。
func ListCompatKeys(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := resolveTenantID(c)
		var keys []models.CompatKey
		if err := a.DB.Where("tenant_id = ?", tenantID).Order("id DESC").Find(&keys).Error; err != nil {
			response.ServerError(c, "查询失败")
			return
		}
		response.OK(c, gin.H{"data": keys, "total": len(keys), "success": true})
	}
}

// CreateCompatKey 新建兼容 key。
func CreateCompatKey(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createCompatKeyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误")
			return
		}
		if req.Source != models.CompatSourceServerChanV1 && req.Source != models.CompatSourceServerChanV2 {
			response.BadRequest(c, "source 必须是 serverchan_v1 或 serverchan_v2")
			return
		}
		externalKey := req.ExternalKey
		if externalKey == "" {
			externalKey = genExternalKey()
		}
		if req.DefaultChannel == "" {
			req.DefaultChannel = "webhook"
		}
		key := models.CompatKey{
			TenantID:       resolveTenantID(c),
			ExternalKey:    externalKey,
			Source:         req.Source,
			DefaultChannel: req.DefaultChannel,
			Description:    req.Description,
			Status:         models.StatusActive,
		}
		if err := a.DB.Create(&key).Error; err != nil {
			response.Conflict(c, "创建失败, key 可能已存在")
			return
		}
		response.OK(c, key)
	}
}

// DeleteCompatKey 删除兼容 key。
func DeleteCompatKey(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的 key ID")
			return
		}
		if err := a.DB.Where("id = ? AND tenant_id = ?", id, resolveTenantID(c)).Delete(&models.CompatKey{}).Error; err != nil {
			response.ServerError(c, "删除失败")
			return
		}
		response.OK(c, gin.H{"ok": true})
	}
}

// genExternalKey 生成随机外部 key。
func genExternalKey() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, 32)
	randRead(buf)
	for i := range buf {
		buf[i] = chars[int(buf[i])%len(chars)]
	}
	return string(buf)
}

// resolveTenantID 从上下文解析归属用户 ID(JWT 用户自身或 API Key 所属用户)。
func resolveTenantID(c *gin.Context) uint64 {
	if u := middleware.CurrentUser(c); u != nil {
		return u.ID
	}
	return middleware.CurrentTenantID(c)
}
