package handler

import (
	"errors"
	"strconv"
	"time"

	"messagepusher/internal/app"
	"messagepusher/internal/pkg/response"
	"messagepusher/internal/service"
	"messagepusher/internal/store"

	"github.com/gin-gonic/gin"
)

// SendMessageRequest 发送消息请求体(对外)。
type SendMessageRequest struct {
	RequestID    string             `json:"request_id"`
	TemplateCode string             `json:"template_code"`
	Title        string             `json:"title"`
	Content      string             `json:"content"`
	Channel      string             `json:"channel"`
	Priority     string             `json:"priority"`
	Recipients   []service.Recipient `json:"recipients" binding:"required,min=1"`
}

// SendMessage 受理发送请求(202 语义)。
func SendMessage(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req SendMessageRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		if req.TemplateCode == "" && req.Content == "" {
			response.BadRequest(c, "template_code 与 content 至少提供一个")
			return
		}

		sendReq := &service.SendRequest{
			RequestID:    req.RequestID,
			TemplateCode: req.TemplateCode,
			Title:        req.Title,
			Content:      req.Content,
			Channel:      req.Channel,
			Priority:     req.Priority,
			Recipients:   req.Recipients,
		}
		tenantID := resolveTenantID(c)
		result, err := a.Messages.Send(c.Request.Context(), tenantID, sendReq)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrRateLimited):
				response.RateLimited(c, "请求频率超限, 请稍后再试")
			case errors.Is(err, service.ErrTemplateMissing):
				response.BadRequest(c, "模板不存在或不可用")
			case errors.Is(err, service.ErrEmptyRecipients):
				response.BadRequest(c, "收件人不能为空")
			default:
				response.ServerError(c, "发送失败: "+err.Error())
			}
			return
		}
		// 202 Accepted: 受理成功。
		c.JSON(202, gin.H{
			"code": response.CodeOK, "message": "ok",
			"data": result,
		})
	}
}

// GetMessage 查询单条消息。
func GetMessage(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的消息 ID")
			return
		}
		m, err := a.Messages.GetMessage(c.Request.Context(), resolveTenantID(c), id)
		if err != nil {
			response.NotFound(c, "消息不存在")
			return
		}
		response.OK(c, m)
	}
}

// QueryMessages 分页查询消息(AntD Pro 分页约定)。
func QueryMessages(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
		size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		f := messageFilterFromQuery(c)
		list, total, err := a.Messages.QueryMessages(c.Request.Context(), resolveTenantID(c), f, page, size)
		if err != nil {
			response.ServerError(c, "查询失败")
			return
		}
		response.OK(c, gin.H{"data": list, "total": total, "success": true})
	}
}

func messageFilterFromQuery(c *gin.Context) store.MessageFilter {
	f := store.MessageFilter{
		Channel:   c.Query("channel"),
		Status:    c.Query("status"),
		Recipient: c.Query("recipient"),
	}
	if s := c.Query("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			f.Since = &t
		}
	}
	if u := c.Query("until"); u != "" {
		if t, err := time.Parse(time.RFC3339, u); err == nil {
			f.Until = &t
		}
	}
	return f
}
