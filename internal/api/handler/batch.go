package handler

import (
	"strconv"

	"axmipusher/internal/app"
	"axmipusher/internal/pkg/response"
	"axmipusher/internal/service"

	"github.com/gin-gonic/gin"
)

// createBatchRequest 创建批量任务请求。
type createBatchRequest struct {
	Name         string              `json:"name" binding:"required"`
	TemplateCode string              `json:"template_code"`
	Title        string              `json:"title"`
	Content      string              `json:"content"`
	Channel      string              `json:"channel"`
	Priority     string              `json:"priority"`
	Recipients   []service.Recipient `json:"recipients" binding:"required,min=1"`
}

// CreateBatchTask 创建批量任务。
func CreateBatchTask(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createBatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		cfg := &service.BatchTaskConfig{
			TemplateCode: req.TemplateCode,
			Title:        req.Title,
			Content:      req.Content,
			Channel:      req.Channel,
			Priority:     req.Priority,
			Recipients:   req.Recipients,
		}
		task, err := a.Batch.Create(c.Request.Context(), resolveTenantID(c), req.Name, cfg)
		if err != nil {
			response.BadRequest(c, "创建失败: "+err.Error())
			return
		}
		Audit(a.DB, c, currentUserID(c), currentUserEmail(c), "batch.create",
			gin.H{"task_id": task.ID, "total": task.Total})
		response.OK(c, task)
	}
}

// ListBatchTasks 任务列表。
func ListBatchTasks(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
		size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		list, total, err := a.Batch.List(resolveTenantID(c), page, size)
		if err != nil {
			response.ServerError(c, "查询失败")
			return
		}
		response.OK(c, gin.H{"data": list, "total": total, "success": true})
	}
}

// GetBatchTask 任务详情。
func GetBatchTask(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的任务 ID")
			return
		}
		task, err := a.Batch.Get(resolveTenantID(c), id)
		if err != nil {
			response.NotFound(c, "任务不存在")
			return
		}
		response.OK(c, task)
	}
}

// CancelBatchTask 取消任务。
func CancelBatchTask(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的任务 ID")
			return
		}
		if err := a.Batch.Cancel(resolveTenantID(c), id); err != nil {
			response.BadRequest(c, "取消失败: "+err.Error())
			return
		}
		Audit(a.DB, c, currentUserID(c), currentUserEmail(c), "batch.cancel", gin.H{"task_id": id})
		response.OK(c, gin.H{"ok": true})
	}
}
