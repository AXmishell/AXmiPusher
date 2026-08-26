package handler

import (
	"strconv"

	"axmipusher/internal/app"
	"axmipusher/internal/models"
	"axmipusher/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// planRequest 套餐请求。
type planRequest struct {
	Name         string  `json:"name" binding:"required"`
	Price        float64 `json:"price"`
	DurationDays int     `json:"duration_days"`
	Quota        string  `json:"quota"` // JSON 字符串
	Description  string  `json:"description"`
	Status       string  `json:"status"`
	SortOrder    int     `json:"sort_order"`
}

// ListPlans 套餐列表。
func ListPlans(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var plans []models.Plan
		if err := a.DB.Order("sort_order ASC, id ASC").Find(&plans).Error; err != nil {
			response.ServerError(c, "查询失败")
			return
		}
		response.OK(c, gin.H{"data": plans, "total": len(plans), "success": true})
	}
}

// CreatePlan 创建套餐。
func CreatePlan(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req planRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		plan := models.Plan{
			Name: req.Name, Price: req.Price, DurationDays: req.DurationDays,
			Quota: req.Quota, Description: req.Description,
			Status: defaultStr(req.Status, models.StatusActive), SortOrder: req.SortOrder,
		}
		if err := a.DB.Create(&plan).Error; err != nil {
			response.ServerError(c, "创建失败")
			return
		}
		Audit(a.DB, c, currentAdminID(c), currentAdminEmail(c), "plan.create", gin.H{"id": plan.ID, "name": plan.Name})
		response.OK(c, plan)
	}
}

// UpdatePlan 更新套餐。
func UpdatePlan(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的套餐 ID")
			return
		}
		var req planRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误")
			return
		}
		updates := map[string]any{
			"name": req.Name, "price": req.Price, "duration_days": req.DurationDays,
			"quota": req.Quota, "description": req.Description, "sort_order": req.SortOrder,
		}
		if req.Status != "" {
			updates["status"] = req.Status
		}
		if err := a.DB.Model(&models.Plan{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			response.ServerError(c, "更新失败")
			return
		}
		Audit(a.DB, c, currentAdminID(c), currentAdminEmail(c), "plan.update", gin.H{"id": id})
		response.OK(c, gin.H{"ok": true})
	}
}

// DeletePlan 删除套餐。
func DeletePlan(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的套餐 ID")
			return
		}
		if err := a.DB.Delete(&models.Plan{}, id).Error; err != nil {
			response.ServerError(c, "删除失败")
			return
		}
		Audit(a.DB, c, currentAdminID(c), currentAdminEmail(c), "plan.delete", gin.H{"id": id})
		response.OK(c, gin.H{"ok": true})
	}
}

// ListPaymentOrders 支付订单列表。
func ListPaymentOrders(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
		size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		status := c.Query("status")
		if page < 1 {
			page = 1
		}
		if size < 1 || size > 100 {
			size = 20
		}
		q := a.DB.Model(&models.PaymentOrder{})
		if status != "" {
			q = q.Where("status = ?", status)
		}
		var total int64
		q.Count(&total)
		var orders []models.PaymentOrder
		if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&orders).Error; err != nil {
			response.ServerError(c, "查询失败")
			return
		}
		response.OK(c, gin.H{"data": orders, "total": total, "success": true})
	}
}

func defaultStr(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
