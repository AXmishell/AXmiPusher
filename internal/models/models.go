// Package models 定义平台全部业务数据模型(基于 GORM)。
package models

import (
	"time"

	"gorm.io/gorm"
)

// 通用状态常量。
const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
	StatusPending  = "pending"
)

// 用户角色。
const (
	RoleTenantAdmin    = "tenant_admin"   // 租户管理员(可管理本租户)
	RoleTenantUser     = "tenant_user"    // 租户普通用户
	RolePlatformAdmin  = "platform_admin" // 平台超管(旧版, 迁移后不再新产生)
)

// 管理员角色。
const (
	AdminRoleSuper  = "super_admin" // 超管(安装时创建, 可管理管理员)
	AdminRoleNormal = "admin"       // 普通管理员
)

// 消息状态机。
const (
	MsgPending  = "PENDING"  // 排队中
	MsgSending  = "SENDING"  // 发送中
	MsgSuccess  = "SUCCESS"  // 送达
	MsgFailed   = "FAILED"   // 失败
	MsgRetrying = "RETRYING" // 重试中
	MsgDead     = "DEAD"     // 死信(人工处理)
	MsgCancelled = "CANCELLED" // 已取消
)

// 消息事件类型。
const (
	EventCreated = "created"
	EventSending = "sending"
	EventSuccess = "success"
	EventFailed  = "failed"
	EventRetry   = "retry"
	EventDead    = "dead"
)

// 兼容 key 来源。
const (
	CompatSourceServerChanV1 = "serverchan_v1" // /api/sc/{SCKEY}
	CompatSourceServerChanV2 = "serverchan_v2" // /api/sctapi/{SendKey}
)

// Tenant 租户。
type Tenant struct {
	ID         uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       string         `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Status     string         `gorm:"size:16;not null;default:active" json:"status"`
	Quota      string         `gorm:"type:text" json:"quota"` // JSONB: 配额信息(套餐驱动)
	PlanID     uint64         `json:"plan_id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// User 用户(开放注册, 属于某个租户)。
type User struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID     uint64    `gorm:"not null;index" json:"tenant_id"`
	Email        string    `gorm:"size:255;not null;uniqueIndex" json:"email"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Nickname     string    `gorm:"size:64" json:"nickname"`
	Role         string    `gorm:"size:32;not null;default:tenant_user" json:"role"`
	Status       string    `gorm:"size:16;not null;default:active" json:"status"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Admin 平台管理员(独立于用户中心 users 表, 支持多管理员)。
type Admin struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Email        string     `gorm:"size:255;not null;uniqueIndex" json:"email"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	Nickname     string     `gorm:"size:64" json:"nickname"`
	Role         string     `gorm:"size:32;not null;default:super_admin" json:"role"`
	Status       string     `gorm:"size:16;not null;default:active" json:"status"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// APIKey 平台 API Key(服务端调用 /api/v1/* 使用)。
type APIKey struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID   uint64     `gorm:"not null;index" json:"tenant_id"`
	Name       string     `gorm:"size:64;not null" json:"name"`
	KeyPrefix  string     `gorm:"size:16;not null" json:"key_prefix"`  // 展示用前缀
	KeyHash    string     `gorm:"size:128;not null;uniqueIndex" json:"-"` // SHA-256
	Scopes     string     `gorm:"type:text" json:"scopes"`            // JSON 数组
	Status     string     `gorm:"size:16;not null;default:active" json:"status"`
	ExpiresAt  *time.Time `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CompatKey 兼容层 key 映射表(外部 key → 租户)。
type CompatKey struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID       uint64    `gorm:"not null;index" json:"tenant_id"`
	ExternalKey    string    `gorm:"size:255;not null;uniqueIndex" json:"external_key"`
	Source         string    `gorm:"size:32;not null;index" json:"source"` // serverchan_v1 | serverchan_v2
	DefaultChannel string    `gorm:"size:32;default:webhook" json:"default_channel"`
	Description    string    `gorm:"size:255" json:"description"`
	Status         string    `gorm:"size:16;not null;default:active" json:"status"`
	LastUsedAt     *time.Time `json:"last_used_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Template 消息模板。
type Template struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID    uint64    `gorm:"not null;uniqueIndex:uk_tenant_code" json:"tenant_id"`
	Code        string    `gorm:"size:64;not null;uniqueIndex:uk_tenant_code" json:"code"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Content     string    `gorm:"type:text;not null" json:"content"` // 含 {{var}} 占位符
	ChannelType string    `gorm:"size:32;not null;default:webhook" json:"channel_type"`
	Status      string    `gorm:"size:16;not null;default:active" json:"status"` // active | disabled
	CurrentVersion int    `gorm:"not null;default:1" json:"current_version"`
	CreatedBy   uint64    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 模板表。
func (Template) TableName() string { return "templates" }

// TemplateVersion 模板版本(审核流: pending → approved/rejected)。
type TemplateVersion struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TemplateID uint64     `gorm:"not null;index" json:"template_id"`
	Version    int        `gorm:"not null" json:"version"`
	Content    string     `gorm:"type:text;not null" json:"content"`
	ReviewStatus string   `gorm:"size:16;not null;default:pending" json:"review_status"` // pending|approved|rejected
	ReviewNote string     `gorm:"size:512" json:"review_note"`
	ReviewedBy *uint64    `json:"reviewed_by"`
	ReviewedAt *time.Time `json:"reviewed_at"`
	CreatedBy  uint64     `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Channel 渠道配置(邮件 SMTP 等, 租户可覆盖平台默认)。
type Channel struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID  uint64         `gorm:"not null;default:0;index" json:"tenant_id"` // 0 = 平台默认
	Type      string         `gorm:"size:32;not null" json:"type"`              // email|apns|fcm|webhook|inapp
	Name      string         `gorm:"size:64;not null" json:"name"`
	Config    string         `gorm:"type:text" json:"config"` // JSONB: 渠道私有配置(SMTP 等)
	Status    string         `gorm:"size:16;not null;default:active" json:"status"`
	Priority  int            `gorm:"not null;default:0" json:"priority"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// WebhookSubscription 业务方回调订阅(消息状态事件推送)。
type WebhookSubscription struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID  uint64         `gorm:"not null;index" json:"tenant_id"`
	URL       string         `gorm:"size:512;not null" json:"url"`
	Secret    string         `gorm:"size:128" json:"-"`
	Events    string         `gorm:"type:text;not null" json:"events"` // JSON 数组: ["success","failed"]
	Status    string         `gorm:"size:16;not null;default:active" json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// BatchTask 批量任务。
type BatchTask struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID  uint64    `gorm:"not null;index" json:"tenant_id"`
	Name      string    `gorm:"size:128" json:"name"`
	Status    string    `gorm:"size:16;not null;default:pending" json:"status"` // pending|running|done|failed
	Total     int64     `gorm:"not null;default:0" json:"total"`
	Success   int64     `gorm:"not null;default:0" json:"success"`
	Failed    int64     `gorm:"not null;default:0" json:"failed"`
	Config    string    `gorm:"type:text" json:"config"` // JSON: 任务参数
	Error     string    `gorm:"size:1024" json:"error"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Unsubscribe 退订名单(营销消息发送前校验)。
type Unsubscribe struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID  uint64    `gorm:"not null;uniqueIndex:uk_tenant_channel_target" json:"tenant_id"`
	Channel   string    `gorm:"size:32;not null;uniqueIndex:uk_tenant_channel_target" json:"channel"`
	Target    string    `gorm:"size:255;not null;uniqueIndex:uk_tenant_channel_target" json:"target"` // 邮箱/手机号/设备ID
	Reason    string    `gorm:"size:255" json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

// AuditLog 审计日志。
type AuditLog struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID  uint64    `gorm:"index" json:"tenant_id"`
	ActorID   uint64    `gorm:"index" json:"actor_id"`
	ActorEmail string   `gorm:"size:255" json:"actor_email"`
	Action    string    `gorm:"size:64;not null;index" json:"action"`
	Detail    string    `gorm:"type:text" json:"detail"`
	IP        string    `gorm:"size:64" json:"ip"`
	CreatedAt time.Time `json:"created_at"`
}

// Plan 套餐定义。
type Plan struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string    `gorm:"size:64;not null" json:"name"`
	Price        float64   `gorm:"not null;default:0" json:"price"`
	DurationDays int       `gorm:"not null;default:30" json:"duration_days"`
	Quota        string    `gorm:"type:text;not null" json:"quota"` // JSON: {monthly_messages:10000, channels:[...]}
	Description  string    `gorm:"size:512" json:"description"`
	Status       string    `gorm:"size:16;not null;default:active" json:"status"`
	SortOrder    int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Subscription 租户订阅(保留历史, 用 status 区分生效中)。
type Subscription struct {
	ID        uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID  uint64     `gorm:"not null;index" json:"tenant_id"`
	PlanID    uint64     `gorm:"not null" json:"plan_id"`
	StartAt   time.Time  `json:"start_at"`
	EndAt     time.Time  `json:"end_at"`
	Status    string     `gorm:"size:16;not null;default:active" json:"status"` // active|expired
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// PaymentOrder 支付订单(易支付)。
type PaymentOrder struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID     uint64     `gorm:"not null;index" json:"tenant_id"`
	PlanID       uint64     `json:"plan_id"`
	Type         string     `gorm:"size:16;not null;default:alipay" json:"type"` // alipay | wxpay
	OutTradeNo   string     `gorm:"size:64;not null;uniqueIndex" json:"out_trade_no"` // 平台侧订单号
	EpayTradeNo  string     `gorm:"size:64" json:"epay_trade_no"`                     // 易支付订单号
	Amount       float64    `gorm:"not null" json:"amount"`
	Status       string     `gorm:"size:16;not null;default:pending" json:"status"` // pending|paid|closed
	NotifyData   string     `gorm:"type:text" json:"-"`                             // 回调原文
	PaidAt       *time.Time `json:"paid_at"`
	ExpiredAt    *time.Time `json:"expired_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// IdempotencyRecord 幂等记录(tenant_id + request_id 唯一)。
type IdempotencyRecord struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID  uint64    `gorm:"not null;uniqueIndex:uk_tenant_request" json:"tenant_id"`
	RequestID string    `gorm:"size:128;not null;uniqueIndex:uk_tenant_request" json:"request_id"`
	MessageID uint64    `gorm:"not null" json:"message_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Setting 平台运行时设置(键值对, JSON 值)。
// 键: smtp / epay / retention_days / admin_path 等。
type Setting struct {
	Key       string    `gorm:"primaryKey;size:64" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedBy uint64    `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
}

// InappMessage 站内信(渠道 inapp 的收件箱记录)。
type InappMessage struct {
	ID        uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID  uint64     `gorm:"not null;index:idx_inapp_user" json:"tenant_id"`
	UserID    uint64     `gorm:"not null;index:idx_inapp_user" json:"user_id"`
	UserEmail string     `gorm:"size:255;not null" json:"user_email"`
	Title     string     `gorm:"size:255" json:"title"`
	Content   string     `gorm:"type:text" json:"content"`
	IsRead    bool       `gorm:"not null;default:false;index" json:"is_read"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}
