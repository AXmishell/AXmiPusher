package handler

import (
	"crypto/rand"
	"encoding/hex"

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

// CurrentUser 从 gin 上下文取当前登录用户。
func CurrentUser(c *gin.Context) *models.User {
	if v, ok := c.Get(string(middleware.CtxUser)); ok {
		if u, ok := v.(*models.User); ok {
			return u
		}
	}
	return nil
}
