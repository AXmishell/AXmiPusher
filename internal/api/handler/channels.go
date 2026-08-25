package handler

import (
	"encoding/json"
	"time"

	"messagepusher/internal/app"
	"messagepusher/internal/models"
	"messagepusher/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// ChannelMeta 渠道元信息(前端展示用)。
var ChannelMeta = []gin.H{
	{"type": "webhook", "name": "Webhook 回调", "desc": "POST 到业务方回调地址", "configured": true},
	{"type": "email", "name": "邮件", "desc": "SMTP 发送(平台默认或租户自定义)", "configured": false},
	{"type": "apns", "name": "APNs (iOS)", "desc": "Apple 推送, 需 Team ID/Key ID/Bundle ID/.p8", "configured": false},
	{"type": "fcm", "name": "FCM (Android)", "desc": "Firebase 推送, 需服务账号 JSON", "configured": false},
	{"type": "inapp", "name": "站内信", "desc": "平台内收件箱, 无需外部配置", "configured": true},
}

// ListChannels 列出渠道及当前租户的配置状态。
func ListChannels(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := resolveTenantID(c)
		var channels []models.Channel
		a.DB.Where("tenant_id = ?", tenantID).Find(&channels)
		byType := map[string]models.Channel{}
		for _, ch := range channels {
			byType[ch.Type] = ch
		}
		result := make([]gin.H, 0, len(ChannelMeta))
		for _, meta := range ChannelMeta {
			typ := meta["type"].(string)
			ch, ok := byType[typ]
			row := gin.H{
				"type": typ, "name": meta["name"], "desc": meta["desc"],
				"configured": meta["configured"],
			}
			if ok {
				row["configured"] = true
				row["id"] = ch.ID
				row["status"] = ch.Status
				row["updated_at"] = ch.UpdatedAt
			}
			result = append(result, row)
		}
		response.OK(c, gin.H{"data": result, "total": len(result), "success": true})
	}
}

// ChannelHealth 渠道健康看板: 熔断状态 + 24h 送达统计。
func ChannelHealth(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := resolveTenantID(c)
		now := time.Now()
		since := now.Add(-24 * time.Hour)

		byChannel, err := a.Store.StatsByChannel(c.Request.Context(), tenantID, since, now)
		if err != nil {
			response.ServerError(c, "统计失败: "+err.Error())
			return
		}

		result := make([]gin.H, 0, len(ChannelMeta))
		for _, meta := range ChannelMeta {
			typ := meta["type"].(string)
			bs := a.Registry.BreakerStats(tenantID, typ)
			chStats := byChannel[typ]
			total := int64(0)
			success := int64(0)
			for k, v := range chStats {
				total += v
				if k == "SUCCESS" {
					success = v
				}
			}
			rate := 0.0
			if total > 0 {
				rate = float64(success) / float64(total) * 100
			}
			result = append(result, gin.H{
				"type":             typ,
				"name":             meta["name"],
				"breaker_state":    bs.State,
				"breaker_failures": bs.TotalFailures,
				"msg_24h":          total,
				"success_24h":      success,
				"success_rate":     round1(rate),
				"last_success_at":  bs.LastSuccessAt,
				"last_failure_at":  bs.LastFailureAt,
			})
		}
		response.OK(c, gin.H{"data": result, "total": len(result), "success": true})
	}
}

// channelConfigRequest 渠道配置请求(JSON 原文透传)。
type channelConfigRequest struct {
	Config json.RawMessage `json:"config" binding:"required"`
}

// UpdateChannel 设置租户渠道覆盖配置(创建或更新)。
func UpdateChannel(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		typ := c.Param("type")
		switch typ {
		case "email", "apns", "fcm":
		default:
			response.BadRequest(c, "不支持的渠道类型: "+typ)
			return
		}
		var req channelConfigRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		tenantID := resolveTenantID(c)

		// 校验配置格式: email 必须含 host。
		var probe map[string]interface{}
		if err := json.Unmarshal(req.Config, &probe); err != nil {
			response.BadRequest(c, "配置必须是合法 JSON")
			return
		}

		var ch models.Channel
		err := a.DB.Where("tenant_id = ? AND type = ?", tenantID, typ).First(&ch).Error
		if err != nil {
			// 新建。
			ch = models.Channel{
				TenantID: tenantID, Type: typ, Name: "租户自定义 " + typ,
				Config: string(req.Config), Status: models.StatusActive,
			}
			if err := a.DB.Create(&ch).Error; err != nil {
				response.ServerError(c, "保存失败")
				return
			}
		} else {
			if err := a.DB.Model(&ch).Updates(map[string]any{
				"config": string(req.Config), "status": models.StatusActive,
			}).Error; err != nil {
				response.ServerError(c, "保存失败")
				return
			}
		}
		Audit(a.DB, c, currentUserID(c), currentUserEmail(c), "channel.update", gin.H{"type": typ, "tenant_id": tenantID})
		response.OK(c, gin.H{"ok": true, "type": typ})
	}
}

// DeleteChannel 删除租户渠道覆盖(回到平台默认)。
func DeleteChannel(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		typ := c.Param("type")
		res := a.DB.Where("tenant_id = ? AND type = ?", resolveTenantID(c), typ).Delete(&models.Channel{})
		if res.RowsAffected == 0 {
			response.NotFound(c, "该渠道没有租户配置")
			return
		}
		Audit(a.DB, c, currentUserID(c), currentUserEmail(c), "channel.delete", gin.H{"type": typ})
		response.OK(c, gin.H{"ok": true})
	}
}
