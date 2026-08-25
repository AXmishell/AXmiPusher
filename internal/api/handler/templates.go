package handler

import (
	"strconv"

	"messagepusher/internal/app"
	"messagepusher/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// createTemplateRequest 创建模板。
type createTemplateRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Content     string `json:"content" binding:"required"`
	ChannelType string `json:"channel_type"`
}

// ListTemplates 租户模板列表。
func ListTemplates(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
		size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		list, total, err := a.Templates.ListTemplates(resolveTenantID(c), page, size)
		if err != nil {
			response.ServerError(c, "查询失败")
			return
		}
		response.OK(c, gin.H{"data": list, "total": total, "success": true})
	}
}

// CreateTemplate 创建模板(自动提交审核)。
func CreateTemplate(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createTemplateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		if req.ChannelType == "" {
			req.ChannelType = "webhook"
		}
		tpl, err := a.Templates.CreateTemplate(resolveTenantID(c), currentUserID(c), req.Code, req.Name, req.Content, req.ChannelType)
		if err != nil {
			response.BadRequest(c, "创建失败: "+err.Error())
			return
		}
		Audit(a.DB, c, currentUserID(c), currentUserEmail(c), "template.create", gin.H{"id": tpl.ID, "code": req.Code})
		response.OK(c, tpl)
	}
}

// UpdateTemplate 更新模板(生成新版本待审核)。
func UpdateTemplate(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的模板 ID")
			return
		}
		var req createTemplateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误")
			return
		}
		if err := a.Templates.UpdateTemplate(resolveTenantID(c), id, currentUserID(c), req.Name, req.Content, req.ChannelType); err != nil {
			response.BadRequest(c, "更新失败: "+err.Error())
			return
		}
		Audit(a.DB, c, currentUserID(c), currentUserEmail(c), "template.update", gin.H{"id": id})
		response.OK(c, gin.H{"ok": true})
	}
}

// GetTemplate 模板详情。
func GetTemplate(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的模板 ID")
			return
		}
		tpl, err := a.Templates.GetTemplate(resolveTenantID(c), id)
		if err != nil {
			response.NotFound(c, "模板不存在")
			return
		}
		response.OK(c, tpl)
	}
}

// DeleteTemplate 删除模板。
func DeleteTemplate(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的模板 ID")
			return
		}
		if err := a.Templates.DeleteTemplate(resolveTenantID(c), id); err != nil {
			response.ServerError(c, "删除失败")
			return
		}
		Audit(a.DB, c, currentUserID(c), currentUserEmail(c), "template.delete", gin.H{"id": id})
		response.OK(c, gin.H{"ok": true})
	}
}

// currentUserID 当前用户 ID(未登录返回 0)。
func currentUserID(c *gin.Context) uint64 {
	if u := CurrentUser(c); u != nil {
		return u.ID
	}
	return 0
}

// currentUserEmail 当前用户邮箱。
func currentUserEmail(c *gin.Context) string {
	if u := CurrentUser(c); u != nil {
		return u.Email
	}
	return ""
}
