package handler

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"

	"messagepusher/internal/api/middleware"
	"messagepusher/internal/models"

	"github.com/gin-gonic/gin"
)

// randRead 用密码学安全随机源填充 buf。
func randRead(buf []byte) {
	rand.Read(buf)
}

// randomHex 生成 n 字节的十六进制随机串。
func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// detectAdminBase 从 admin 前端构建产物的 index.html 解析实际 base 路径。
// 构建脚本将 base 写死为 /{随机串}/, 若运行时生成不同路径会导致管理后台资源 404。
func detectAdminBase(distDir string) string {
	if distDir == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(distDir, "index.html"))
	if err != nil {
		return ""
	}
	// 匹配 <script src="/xxxx/assets/..."> 或 <link href="/xxxx/assets/...">。
	re := regexp.MustCompile(`(?:src|href)="/([A-Za-z0-9]+)/assets/`)
	m := re.FindSubmatch(data)
	if len(m) < 2 {
		return ""
	}
	return string(m[1])
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
