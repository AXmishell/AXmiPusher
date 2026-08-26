package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"messagepusher/internal/app"

	"github.com/gin-gonic/gin"
)

// registerWebRoutes 生产模式托管前端静态资源:
//   /                    → 用户中心(SPA, 经 NoRoute 兜底, 避免与 /api 路由冲突)
//   /{admin_path}/        → 管理员后台(SPA, base 为 /{admin_path}/)
func registerWebRoutes(r *gin.Engine, userDist, adminDist, adminPath string) {
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
	if adminDist != "" && adminPrefix != "" {
		registerSPA(r, adminPrefix, adminDist)
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
			if adminPrefix != "" && (p == adminPrefix || strings.HasPrefix(p, adminPrefix+"/")) {
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

// registerSPA 注册前缀下的 SPA 静态托管: 命中文件返回文件, 否则回退 index.html。
func registerSPA(r *gin.Engine, prefix, distDir string) {
	routePath := prefix + "/*filepath"
	r.GET(routePath, func(c *gin.Context) {
		p := c.Param("filepath")
		if p == "" || p == "/" {
			p = "/"
		}
		full := filepath.Join(distDir, filepath.Clean(strings.TrimPrefix(p, "/")))
		// 目录 → index.html; 不存在 → SPA 回退。
		if info, err := os.Stat(full); err != nil || info.IsDir() {
			full = filepath.Join(distDir, "index.html")
		}
		c.File(full)
	})
}
