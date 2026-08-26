package handler

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"regexp"

	"axmipusher/internal/api/middleware"
	"axmipusher/internal/models"

	"github.com/gin-gonic/gin"
)

// randRead 用密码学安全随机源填充 buf。
func randRead(buf []byte) {
	rand.Read(buf)
}

// rewriteAdminBase 改写 admin dist index.html 中的 base 路径为 newPath。
// 轮换管理员后台随机路径后, 必须同步改写前端资源前缀, 否则管理后台 404。
func rewriteAdminBase(distDir, oldPath, newPath string) error {
	idx := filepath.Join(distDir, "index.html")
	data, err := os.ReadFile(idx)
	if err != nil {
		return err
	}
	if oldPath != "" {
		re := regexp.MustCompile(`/([A-Za-z0-9]+)/assets/`)
		data = re.ReplaceAll(data, []byte("/"+newPath+"/assets/"))
	}
	return os.WriteFile(idx, data, 0o644)
}

// CurrentUser 从 gin 上下文取当前登录用户。
func CurrentUser(c *gin.Context) *models.User {
	if v, ok := c.Get(string(middleware.CtxUser)); ok {
		if u, ok := v.(*models.User); ok {
			return u
		}
	}
	return nil
}

// CurrentAdmin 从 gin 上下文取当前登录管理员。
func CurrentAdmin(c *gin.Context) *models.Admin {
	if v, ok := c.Get(string(middleware.CtxAdmin)); ok {
		if a, ok := v.(*models.Admin); ok {
			return a
		}
	}
	return nil
}
