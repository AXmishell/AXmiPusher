// Package mail 共享邮件发送能力(低层 SMTP 助手, 无渠道/业务依赖)。
package mail

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// Config SMTP 连接配置。
type Config struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	User      string `json:"user"`
	Password  string `json:"password"`
	From      string `json:"from"`      // 发件人地址
	Recipient string `json:"recipient"` // 默认收件人(通道配置用, 发送时未指定收件人时兜底)
}

// UnmarshalJSON 兼容 port 为数字或字符串。
// 历史踩坑: 用户中心通道配置表单曾用文本输入提交 "465"(字符串), 导致 int 解析失败 → 邮件发送死信。
func (c *Config) UnmarshalJSON(data []byte) error {
	var raw struct {
		Host      string          `json:"host"`
		Port      json.RawMessage `json:"port"`
		User      string          `json:"user"`
		Password  string          `json:"password"`
		From      string          `json:"from"`
		Recipient string          `json:"recipient"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.Host = raw.Host
	c.User = raw.User
	c.Password = raw.Password
	c.From = raw.From
	c.Recipient = raw.Recipient
	if len(raw.Port) > 0 && string(raw.Port) != "null" {
		var n int
		if err := json.Unmarshal(raw.Port, &n); err == nil {
			c.Port = n
			return nil
		}
		var s string
		if err := json.Unmarshal(raw.Port, &s); err == nil {
			if p, err2 := strconv.Atoi(strings.TrimSpace(s)); err2 == nil {
				c.Port = p
			}
		}
	}
	return nil
}

// Send 发送邮件。
// 端口 465 走隐式 TLS, 587/25 走 STARTTLS(若支持)。
func Send(cfg Config, to []string, subject, body string) error {
	if cfg.Host == "" {
		return errors.New("SMTP 主机未配置")
	}
	if cfg.Port <= 0 {
		cfg.Port = 465
	}
	from := cfg.From
	if from == "" {
		from = cfg.User
	}
	if from == "" {
		return errors.New("发件人未配置")
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	var conn net.Conn
	var err error
	if cfg.Port == 465 {
		conn, err = tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.Host, InsecureSkipVerify: false})
	} else {
		conn, err = net.DialTimeout("tcp", addr, 10*time.Second)
	}
	if err != nil {
		return fmt.Errorf("连接 SMTP %s 失败: %w", addr, err)
	}
	// 会话总超时: 防止对端非 SMTP 服务(只收不发 banner)导致 worker 永久阻塞。
	sessionDeadline := time.Now().Add(15 * time.Second)
	conn.SetDeadline(sessionDeadline)
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("SMTP 握手失败: %w", err)
	}
	defer client.Close()

	// 587/25: 尝试 STARTTLS。
	if cfg.Port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: cfg.Host}); err != nil {
				return fmt.Errorf("STARTTLS 失败: %w", err)
			}
		}
	}

	// 认证(有账号时)。
	if cfg.User != "" {
		auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP 认证失败: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM 失败: %w", err)
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("RCPT TO %s 失败: %w", rcpt, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA 失败: %w", err)
	}
	msg := BuildMessage(from, to, subject, body)
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("写入邮件内容失败: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("结束 DATA 失败: %w", err)
	}
	return client.Quit()
}

// BuildMessage 构造 RFC5322 邮件(带必要头)。
func BuildMessage(from string, to []string, subject, body string) string {
	var sb strings.Builder
	sb.WriteString("From: " + from + "\r\n")
	sb.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	sb.WriteString("Subject: " + mimeHeader(subject) + "\r\n")
	sb.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("Content-Transfer-Encoding: base64\r\n")
	sb.WriteString("\r\n")
	// base64 正文(保证中文 UTF-8 安全)。
	sb.WriteString(base64Body(body))
	return sb.String()
}

// mimeHeader 对含非 ASCII 的主题做 MIME 编码。
func mimeHeader(s string) string {
	if isASCII(s) {
		return s
	}
	return "=?UTF-8?B?" + base64Encode(s) + "?="
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func base64Body(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
