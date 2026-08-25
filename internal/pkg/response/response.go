// Package response 统一 API 响应格式。
// 成功: {"code":0,"message":"ok","data":{...}}
// 失败: {"code":<业务码>,"message":"<错误信息>","data":null}
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Body 统一响应体。
type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 业务错误码。
const (
	CodeOK            = 0
	CodeBadRequest    = 40000 // 参数错误
	CodeUnauthorized  = 40100 // 未认证/凭证无效
	CodeForbidden     = 40300 // 无权限
	CodeNotFound      = 40400 // 资源不存在
	CodeRateLimited   = 42900 // 限流
	CodeConflict      = 40900 // 冲突(幂等重复等)
	CodeServerError   = 50000 // 服务端错误
)

// OK 成功响应。
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{Code: CodeOK, Message: "ok", Data: data})
}

// Fail 失败响应。
func Fail(c *gin.Context, httpStatus, code int, msg string) {
	c.JSON(httpStatus, Body{Code: code, Message: msg})
}

// BadRequest 参数错误(400)。
func BadRequest(c *gin.Context, msg string) {
	Fail(c, http.StatusBadRequest, CodeBadRequest, msg)
}

// Unauthorized 未认证(401)。
func Unauthorized(c *gin.Context, msg string) {
	Fail(c, http.StatusUnauthorized, CodeUnauthorized, msg)
}

// Forbidden 无权限(403)。
func Forbidden(c *gin.Context, msg string) {
	Fail(c, http.StatusForbidden, CodeForbidden, msg)
}

// NotFound 资源不存在(404)。
func NotFound(c *gin.Context, msg string) {
	Fail(c, http.StatusNotFound, CodeNotFound, msg)
}

// RateLimited 限流(429)。
func RateLimited(c *gin.Context, msg string) {
	Fail(c, http.StatusTooManyRequests, CodeRateLimited, msg)
}

// Conflict 冲突(409)。
func Conflict(c *gin.Context, msg string) {
	Fail(c, http.StatusConflict, CodeConflict, msg)
}

// ServerError 服务端错误(500)。
func ServerError(c *gin.Context, msg string) {
	Fail(c, http.StatusInternalServerError, CodeServerError, msg)
}
