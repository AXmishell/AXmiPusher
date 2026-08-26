package handler

import (
	"time"

	"axmipusher/internal/app"
	"axmipusher/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// StatsMessages 消息统计(按状态)。
func StatsMessages(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := resolveTenantID(c)
		now := time.Now()
		since := now.Add(-24 * time.Hour)
		if s := c.Query("since"); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				since = t
			}
		}
		until := now
		if u := c.Query("until"); u != "" {
			if t, err := time.Parse(time.RFC3339, u); err == nil {
				until = t
			}
		}
		stats, err := a.Store.StatsByStatus(c.Request.Context(), tenantID, since, until)
		if err != nil {
			response.ServerError(c, "统计失败: "+err.Error())
			return
		}
		response.OK(c, gin.H{
			"since":  since,
			"until":  until,
			"status": stats,
		})
	}
}

// StatsOverview 概览统计(发送总量/成功率/渠道分布)。
func StatsOverview(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := resolveTenantID(c)
		now := time.Now()
		since := now.Add(-24 * time.Hour)
		if s := c.Query("since"); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				since = t
			}
		}
		stats, err := a.Store.StatsByStatus(c.Request.Context(), tenantID, since, now)
		if err != nil {
			response.ServerError(c, "统计失败: "+err.Error())
			return
		}
		total := int64(0)
		success := int64(0)
		for k, v := range stats {
			total += v
			if k == "SUCCESS" {
				success = v
			}
		}
		rate := 0.0
		if total > 0 {
			rate = float64(success) / float64(total) * 100
		}
		response.OK(c, gin.H{
			"total":     total,
			"success":   success,
			"failed":    stats["FAILED"] + stats["DEAD"],
			"success_rate": round1(rate),
			"period":    gin.H{"since": since, "until": now},
		})
	}
}

func round1(f float64) float64 {
	return float64(int(f*10+0.5)) / 10
}
