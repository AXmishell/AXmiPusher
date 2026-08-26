package api

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"axmipusher/internal/app"

	"github.com/gin-gonic/gin"
)

// adminPathLikeRe 与 config.ValidateAdminPath 一致的"疑似后台路径"形状(8-32 位纯字母数字)。
// 用于 NoRoute 识别轮换/重启后失去显式路由的旧后台前缀, 直接 404 废除。
var adminPathLikeRe = regexp.MustCompile(`^[A-Za-z0-9]{8,32}$`)

// userCenterRoutes 用户中心 SPA 中为纯字母数字且长度 ≥8 的顶层路由。
// 这些是合法用户中心页面(会被误判为旧后台前缀), 需放行; 其余路由(login/plans/inbox 等)长度 <8
// 或含连字符(api-keys/batch-tasks 等), 不受 adminPathLikeRe 影响。
var userCenterRoutes = map[string]bool{
	"register": true,
	"messages": true,
	"channels": true,
	"settings": true,
}

// registerWebRoutes 托管前端静态资源:
//   /                    → 用户中心(SPA, 经 NoRoute 兜底, 避免与 /api 路由冲突)
//   /{admin_path}/        → 管理员后台(SPA, base 为 /{admin_path}/)
func registerWebRoutes(r *gin.Engine, a *app.App) {
	userDist := a.Cfg.Web.UserDist
	adminDist := a.Cfg.Web.AdminDist
	adminPath := a.Cfg.Admin.RandomPath
	adminPrefix := ""
	if adminPath != "" {
		adminPrefix = "/" + strings.Trim(adminPath, "/")
	} else if adminDist != "" {
		// 未安装(无 config.yaml)时从构建产物解析 base, 保证安装前管理后台路由已注册。
		if p := app.DetectAdminBase(adminDist); p != "" {
			adminPrefix = "/" + p
		}
	}

	// 管理员后台: 显式前缀路由(唯一前缀, 无路由树冲突)。
	// handler 动态校验当前配置路径: 旧前缀(轮换后)直接 404 废除, 不做重定向。
	if adminDist != "" && adminPrefix != "" {
		registerSPA(r, adminPrefix, a)
	}

	// 用户中心: NoRoute 兜底(静态资源优先, 其余回退 index.html)。
	if userDist != "" {
		r.NoRoute(func(c *gin.Context) {
			p := c.Request.URL.Path
			// 排除 API 与安装路由(返回真实 404)。
			// 注意: 必须精确匹配 /api 或 /api/ 前缀, 不能简单 HasPrefix("/api"),
			// 否则 /api-keys 等 SPA 前端路由会被误伤为 404。
			if p == "/api" || strings.HasPrefix(p, "/api/") || p == "/install" {
				c.Status(http.StatusNotFound)
				return
			}
			// 旧后台路径废除(防重启后旧前缀落入用户中心 SPA):
			// 顶层段为 8-32 位纯字母数字 且 不是当前后台路径 且 不是用户中心 SPA 路由 → 404。
			// 运行中轮换的旧前缀由 AdminSPAHandler 404, 重启后丢失显式路由的旧前缀由这里兜底。
			seg := strings.TrimPrefix(p, "/")
			if i := strings.IndexByte(seg, '/'); i >= 0 {
				seg = seg[:i]
			}
			curPath := strings.Trim(a.Cfg.Admin.RandomPath, "/")
			if curPath != "" && seg != curPath && adminPathLikeRe.MatchString(seg) && !userCenterRoutes[seg] {
				c.Status(http.StatusNotFound)
				return
			}
			// 静态文件优先。
			full := filepath.Join(userDist, filepath.Clean(strings.TrimPrefix(p, "/")))
			if info, err := os.Stat(full); err == nil && !info.IsDir() {
				c.File(full)
				return
			}
			// SPA 回退。
			c.File(filepath.Join(userDist, "index.html"))
		})
	}
}

// registerSPA 注册前缀下的 SPA 静态托管(动态校验当前路径, 旧前缀 404 废除)。
func registerSPA(r *gin.Engine, prefix string, a *app.App) {
	routePath := prefix + "/*filepath"
	r.GET(routePath, app.AdminSPAHandler(a, a.Cfg.Web.AdminDist))
}
