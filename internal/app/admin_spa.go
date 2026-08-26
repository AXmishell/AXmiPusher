package app

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// AdminSPAHandler 返回管理后台 SPA 静态托管 handler。
// 请求前缀与当前配置路径一致 → 提供静态文件(命中文件返回文件, 否则回退 index.html);
// 请求前缀与当前配置路径不一致(轮换/改自定义后的旧前缀或未知前缀)→ 404 直接废除。
// 效果: 后台路径必须唯一, 旧路径立即失效且不可再访问, 不做任何重定向(避免旧地址被继续传播)。
func AdminSPAHandler(a *App, distDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cur := "/" + strings.Trim(a.Cfg.Admin.RandomPath, "/")
		reqPath := c.Request.URL.Path
		seg := strings.TrimPrefix(reqPath, "/")
		if i := strings.IndexByte(seg, '/'); i >= 0 {
			seg = seg[:i]
		}
		reqPrefix := "/" + seg
		// 当前配置路径已知, 且请求前缀不一致 → 旧路径/未知前缀直接 404 废除(不重定向)。
		if cur != "/" && reqPrefix != cur {
			c.Status(http.StatusNotFound)
			return
		}
		fp := c.Param("filepath")
		if fp == "" || fp == "/" {
			fp = "/"
		}
		full := filepath.Join(distDir, filepath.Clean(strings.TrimPrefix(fp, "/")))
		if info, err := os.Stat(full); err != nil || info.IsDir() {
			full = filepath.Join(distDir, "index.html")
		}
		c.File(full)
	}
}
