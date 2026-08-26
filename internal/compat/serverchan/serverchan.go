// Package serverchan 实现 Server酱 兼容接口(Server酱3 sc 形态 / Server酱·Turbo版 sctapi 形态)。
// 老客户端零改动接入: 外部 key 查映射表 → 转核心消息 → 复用核心链路。
// 响应格式严格照抄原版, 客户端才能无感切换。
package serverchan

import (
	"net/http"
	"strings"
	"time"

	"axmipusher/internal/app"
	"axmipusher/internal/models"
	"axmipusher/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// parseCompatKey 从 wildcard 段解析 key: "/SCTxxx.send" → "SCTxxx"。
// 兼容带/不带 .send 后缀的请求。
func parseCompatKey(raw string) string {
	key := strings.TrimPrefix(raw, "/")
	key = strings.TrimSuffix(key, ".send")
	key = strings.TrimSpace(key)
	if key == "" || key == "/" {
		return ""
	}
	return key
}

// RegisterRoutes 注册兼容路由(严格照抄原版路径形态)。
//   POST /api/sctapi/{SendKey}.send   Server酱·Turbo版 (sctapi.ftqq.com 形态)
//   GET  /api/sc/{SCKEY}.send         Server酱3 (sc.ftqq.com 形态)
// 兼容实现也用 wildcard 段, 手动解析 {key}.send。
func RegisterRoutes(r *gin.Engine, a *app.App) {
	g := r.Group("/api")
	{
		g.POST("/sctapi/*key", func(c *gin.Context) {
			handle(c, a, models.CompatSourceServerChanV2)
		})
		g.GET("/sc/*key", func(c *gin.Context) {
			handle(c, a, models.CompatSourceServerChanV1)
		})
	}
}

// handle 统一处理兼容请求。
func handle(c *gin.Context, a *app.App, source string) {
	key := parseCompatKey(c.Param("key"))
	if key == "" {
		sendFail(c, source, "invalid key")
		return
	}

	// 1. 查映射表: 外部 key → 租户。
	var compat models.CompatKey
	err := a.DB.Where("external_key = ? AND source = ? AND status = ?",
		key, source, models.StatusActive).First(&compat).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			sendFail(c, source, "invalid key")
			return
		}
		sendFail(c, source, "server error")
		return
	}

	// 2. 解析参数(v1: text+desp, v2: title+desp+short)。
	title, desp := extractParams(c, source)
	if title == "" && desp == "" {
		sendFail(c, source, "empty title or desp")
		return
	}

	now := time.Now()
	// 3. 转换并复用核心链路(建记录 + 入队)。
	mid, err := a.Messages.Send(c.Request.Context(), compat.TenantID, &service.SendRequest{
		Title:      title,
		Content:    desp,
		Channel:    compat.DefaultChannel,
		Priority:   "normal",
		Recipients: []service.Recipient{{Target: key}},
	})
	if err != nil {
		sendFail(c, source, "push failed")
		return
	}

	// 4. 更新 last_used_at。
	a.DB.Model(&compat).Update("last_used_at", &now)

	// 5. 返回原版格式。
	c.JSON(http.StatusOK, successBody(source, mid.MessageID))
}

// extractParams 按版本解析参数。
func extractParams(c *gin.Context, source string) (title, desp string) {
	if source == models.CompatSourceServerChanV2 {
		// v2: application/x-www-form-urlencoded, title + desp + short。
		title = c.PostForm("title")
		desp = c.PostForm("desp")
		if s := c.PostForm("short"); s != "" && title == "" {
			title = s
		}
		return title, desp
	}
	// v1: text(标题) + desp(内容), 兼容 form 与 query。
	title = c.PostForm("text")
	if title == "" {
		title = c.Query("text")
	}
	desp = c.PostForm("desp")
	if desp == "" {
		desp = c.Query("desp")
	}
	return title, desp
}

// successBody 原版成功响应。
// v1: {"errno":0,"errmsg":"success","data":{...}}
// v2: {"code":0,"message":"","data":{"pushid":"...","readkey":"..."}}
func successBody(source string, pushID uint64) map[string]interface{} {
	if source == models.CompatSourceServerChanV1 {
		return map[string]interface{}{
			"errno":  0,
			"errmsg": "success",
			"data":   map[string]interface{}{"pushid": pushID},
		}
	}
	return map[string]interface{}{
		"code":    0,
		"message": "",
		"data": map[string]interface{}{
			"pushid":  pushID,
			"readkey": "",
		},
	}
}

// sendFail 原版失败响应。
func sendFail(c *gin.Context, source, msg string) {
	if source == models.CompatSourceServerChanV1 {
		c.JSON(http.StatusOK, map[string]interface{}{
			"errno":  40001,
			"errmsg": msg,
		})
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{
		"code":    40001,
		"message": msg,
	})
}
