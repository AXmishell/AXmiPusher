package handler

import (
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"

	"axmipusher/internal/app"
	"axmipusher/internal/config"
	"axmipusher/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// GetSettings 获取平台设置(敏感字段脱敏)。
func GetSettings(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		// SMTP(隐藏密码)。
		smtp := map[string]interface{}{}
		if raw, err := a.Settings.Get("smtp", ""); err == nil && raw != "" {
			json.Unmarshal([]byte(raw), &smtp)
			if _, ok := smtp["password"]; ok {
				smtp["password"] = ""
			}
		}
		// 易支付(隐藏 key)。
		epay := map[string]interface{}{}
		if raw, err := a.Settings.Get("epay", ""); err == nil && raw != "" {
			json.Unmarshal([]byte(raw), &epay)
			if _, ok := epay["key"]; ok {
				epay["key"] = ""
			}
		}
		retention, _ := a.Settings.Get("retention_days", "90")
		adminPath := a.Cfg.Admin.RandomPath
		rateLimit, _ := a.Settings.Get("rate_limit_per_minute", "600")
		response.OK(c, gin.H{
			"smtp":            smtp,
			"epay":            epay,
			"retention_days":  retention,
			"admin_path":      adminPath,
			"rate_limit_per_minute": rateLimit,
		})
	}
}

// UpdateSettings 更新平台设置。
func UpdateSettings(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			SMTP           *json.RawMessage `json:"smtp"`
			Epay           *json.RawMessage `json:"epay"`
			RetentionDays  *int             `json:"retention_days"`
			RateLimitPerMinute *int         `json:"rate_limit_per_minute"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		if req.SMTP != nil {
			if err := a.Settings.Set("smtp", string(*req.SMTP), currentAdminID(c)); err != nil {
				response.ServerError(c, "保存 SMTP 配置失败")
				return
			}
		}
		if req.Epay != nil {
			if err := a.Settings.Set("epay", string(*req.Epay), currentAdminID(c)); err != nil {
				response.ServerError(c, "保存易支付配置失败")
				return
			}
		}
		if req.RetentionDays != nil {
			if err := a.Settings.Set("retention_days", strconv.Itoa(*req.RetentionDays), currentAdminID(c)); err != nil {
				response.ServerError(c, "保存失败")
				return
			}
			a.Cfg.Retention.MessageDays = *req.RetentionDays
		}
		if req.RateLimitPerMinute != nil {
			a.Settings.Set("rate_limit_per_minute", strconv.Itoa(*req.RateLimitPerMinute), currentAdminID(c))
			// 立即生效(限流器支持动态调整额度)。
			a.Cfg.RateLimit.PerMinute = *req.RateLimitPerMinute
			a.Limiter.SetPerMinute(*req.RateLimitPerMinute)
		}
		Audit(a.DB, c, currentAdminID(c), currentAdminEmail(c), "settings.update", gin.H{})
		response.OK(c, gin.H{"ok": true})
	}
}

// RotateAdminPath 轮换/自定义管理员后台路径。
// 请求体可选字段 admin_path: 填写则校验后使用(自定义路径), 留空则随机生成(轮换)。
// 旧路径立即废除(404), 后台路径全局唯一。
func RotateAdminPath(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			AdminPath string `json:"admin_path"` // 留空 = 随机轮换
		}
		// 兼容空 body(纯轮换)。
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		oldPath := strings.Trim(a.Cfg.Admin.RandomPath, "/")
		// 生成/校验新路径。
		newPath := strings.TrimSpace(req.AdminPath)
		if newPath != "" {
			// 自定义路径: 校验(字符集/长度/保留字)+ 唯一性(不得与当前路径相同)。
			var err error
			newPath, err = config.ValidateAdminPath(newPath)
			if err != nil {
				response.BadRequest(c, "后台路径不合法: "+err.Error())
				return
			}
			if newPath == oldPath {
				response.BadRequest(c, "新路径与当前路径相同, 请更换")
				return
			}
		} else {
			// 随机生成 16 位大小写字母数字新路径并写入配置(持久化)。
			var err error
			newPath, err = config.GenerateRandomAdminPath()
			if err != nil {
				response.ServerError(c, "生成随机路径失败")
				return
			}
		}
		// 同步改写 admin dist index.html 的资源前缀(绝对 base 构建时需要, 相对 base 无副作用)。
		if a.Cfg.Web.AdminDist != "" {
			if err := rewriteAdminBase(a.Cfg.Web.AdminDist, oldPath, newPath); err != nil {
				response.ServerError(c, "改写前端 base 失败: "+err.Error())
				return
			}
		}
		// 动态注册新前缀路由(立即生效); 旧前缀路由保留但 handler 校验前缀不一致时 404 废除。
		a.RegisterAdminSPA(newPath, a.Cfg.Web.AdminDist)
		a.Cfg.Admin.RandomPath = newPath
		if err := a.Cfg.Save(); err != nil {
			response.ServerError(c, "保存配置失败")
			return
		}
		Audit(a.DB, c, currentAdminID(c), currentAdminEmail(c), "settings.rotate_admin_path",
			gin.H{"new_path": newPath, "custom": req.AdminPath != ""})
		response.OK(c, gin.H{"admin_path": newPath})
	}
}
