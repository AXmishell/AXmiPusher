package handler

import (
	"strconv"

	"messagepusher/internal/app"
	"messagepusher/internal/models"
	"messagepusher/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// ListReviews 待审核模板列表(默认 pending)。
func ListReviews(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.DefaultQuery("review_status", models.StatusPending)
		page, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
		size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		items, total, err := a.Templates.ListReviews(status, page, size)
		if err != nil {
			response.ServerError(c, "查询失败")
			return
		}
		response.OK(c, gin.H{"data": items, "total": total, "success": true})
	}
}

// ApproveReview 批准模板版本。
func ApproveReview(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		templateID, _ := strconv.ParseUint(c.Param("templateId"), 10, 64)
		versionID, _ := strconv.ParseUint(c.Param("versionId"), 10, 64)
		if templateID == 0 || versionID == 0 {
			response.BadRequest(c, "无效的参数")
			return
		}
		var req struct {
			Note string `json:"note"`
		}
		c.ShouldBindJSON(&req)
		if err := a.Templates.ApproveVersion(templateID, versionID, currentUserID(c), req.Note); err != nil {
			response.BadRequest(c, "批准失败: "+err.Error())
			return
		}
		Audit(a.DB, c, currentUserID(c), currentUserEmail(c), "review.approve",
			gin.H{"template_id": templateID, "version_id": versionID})
		response.OK(c, gin.H{"ok": true})
	}
}

// RejectReview 驳回模板版本。
func RejectReview(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		templateID, _ := strconv.ParseUint(c.Param("templateId"), 10, 64)
		versionID, _ := strconv.ParseUint(c.Param("versionId"), 10, 64)
		if templateID == 0 || versionID == 0 {
			response.BadRequest(c, "无效的参数")
			return
		}
		var req struct {
			Note string `json:"note"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "请填写驳回原因")
			return
		}
		if err := a.Templates.RejectVersion(templateID, versionID, currentUserID(c), req.Note); err != nil {
			response.BadRequest(c, "驳回失败: "+err.Error())
			return
		}
		Audit(a.DB, c, currentUserID(c), currentUserEmail(c), "review.reject",
			gin.H{"template_id": templateID, "version_id": versionID, "note": req.Note})
		response.OK(c, gin.H{"ok": true})
	}
}
