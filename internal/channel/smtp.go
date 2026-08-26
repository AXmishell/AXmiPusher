package channel

import "axmipusher/internal/pkg/mail"

// SMTPConfig 邮件连接配置(别名, 与 pkg/mail.Config 共享)。
type SMTPConfig = mail.Config

// smtpSend 发送邮件(实现已抽到 pkg/mail, 此处保持渠道包内兼容)。
func smtpSend(cfg SMTPConfig, to []string, subject, body string) error {
	return mail.Send(cfg, to, subject, body)
}

// buildMessage 构造 RFC5322 邮件(实现已抽到 pkg/mail)。
func buildMessage(from string, to []string, subject, body string) string {
	return mail.BuildMessage(from, to, subject, body)
}
