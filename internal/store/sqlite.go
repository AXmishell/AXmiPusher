package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// storedMessage 消息记录表(GORM 模型, 本地模式复用业务库)。
type storedMessage struct {
	MessageID  uint64    `gorm:"primaryKey;autoIncrement" json:"message_id"`
	TenantID   uint64    `gorm:"not null;index:idx_tenant_created" json:"tenant_id"`
	RequestID  string    `gorm:"size:128;not null;index" json:"request_id"`
	Channel    string    `gorm:"size:32;not null;index" json:"channel"`
	Title      string    `gorm:"size:255" json:"title"`
	Content    string    `gorm:"type:text" json:"content"`
	Recipient  string    `gorm:"size:255;index" json:"recipient"`
	Status     string    `gorm:"size:16;not null;index" json:"status"`
	Error      string    `gorm:"size:1024" json:"error"`
	RetryCount int       `gorm:"not null;default:0" json:"retry_count"`
	CostMs     int64     `gorm:"not null;default:0" json:"cost_ms"`
	CreatedAt  time.Time `gorm:"index:idx_tenant_created" json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// storedMessageEvent 消息事件表。
type storedMessageEvent struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	MessageID uint64    `gorm:"not null;index" json:"message_id"`
	EventType string    `gorm:"size:32;not null" json:"event_type"`
	Detail    string    `gorm:"size:1024" json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 消息记录表名。
func (storedMessage) TableName() string { return "messages" }

// TableName 事件表名。
func (storedMessageEvent) TableName() string { return "message_events" }

// SQLiteStore 基于 GORM(本地模式, 复用业务 SQLite 库)的消息存储。
type SQLiteStore struct {
	db *gorm.DB
}

// NewSQLiteStore 创建 SQLite 消息存储, 并迁移消息表。
func NewSQLiteStore(db *gorm.DB) (*SQLiteStore, error) {
	if err := db.AutoMigrate(&storedMessage{}, &storedMessageEvent{}); err != nil {
		return nil, fmt.Errorf("迁移消息表失败: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

// SaveMessage 创建消息记录。
func (s *SQLiteStore) SaveMessage(ctx context.Context, m *Message) (uint64, error) {
	rec := toStored(m)
	if err := s.db.WithContext(ctx).Create(rec).Error; err != nil {
		return 0, err
	}
	m.MessageID = rec.MessageID
	return rec.MessageID, nil
}

// UpdateStatus 更新消息状态。
func (s *SQLiteStore) UpdateStatus(ctx context.Context, messageID uint64, status, errMsg string) error {
	return s.db.WithContext(ctx).Model(&storedMessage{}).
		Where("message_id = ?", messageID).
		Updates(map[string]any{"status": status, "error": errMsg, "updated_at": time.Now()}).Error
}

// GetMessage 查询单条消息。
func (s *SQLiteStore) GetMessage(ctx context.Context, messageID uint64) (*Message, error) {
	var rec storedMessage
	if err := s.db.WithContext(ctx).First(&rec, "message_id = ?", messageID).Error; err != nil {
		return nil, err
	}
	return fromStored(&rec), nil
}

// QueryMessages 分页查询。
func (s *SQLiteStore) QueryMessages(ctx context.Context, tenantID uint64, f MessageFilter, page, size int) ([]*Message, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	q := s.db.WithContext(ctx).Model(&storedMessage{}).Where("tenant_id = ?", tenantID)
	if f.Channel != "" {
		q = q.Where("channel = ?", f.Channel)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Recipient != "" {
		q = q.Where("recipient = ?", f.Recipient)
	}
	if f.Since != nil {
		q = q.Where("created_at >= ?", *f.Since)
	}
	if f.Until != nil {
		q = q.Where("created_at <= ?", *f.Until)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var recs []storedMessage
	if err := q.Order("message_id DESC").Offset((page - 1) * size).Limit(size).Find(&recs).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*Message, 0, len(recs))
	for i := range recs {
		out = append(out, fromStored(&recs[i]))
	}
	return out, total, nil
}

// SaveEvent 记录消息事件。
func (s *SQLiteStore) SaveEvent(ctx context.Context, e *MessageEvent) error {
	return s.db.WithContext(ctx).Create(&storedMessageEvent{
		MessageID: e.MessageID,
		EventType: e.EventType,
		Detail:    e.Detail,
		CreatedAt: e.CreatedAt,
	}).Error
}

// StatsByStatus 按状态统计。
func (s *SQLiteStore) StatsByStatus(ctx context.Context, tenantID uint64, since, until time.Time) (map[string]int64, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	q := s.db.WithContext(ctx).Model(&storedMessage{}).
		Select("status, COUNT(*) as count").
		Where("tenant_id = ? AND created_at >= ? AND created_at <= ?", tenantID, since, until).
		Group("status")
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, r := range rows {
		out[r.Status] = r.Count
	}
	return out, nil
}

// StatsByChannel 按渠道+状态统计。
func (s *SQLiteStore) StatsByChannel(ctx context.Context, tenantID uint64, since, until time.Time) (map[string]map[string]int64, error) {
	type row struct {
		Channel string
		Status  string
		Count   int64
	}
	var rows []row
	q := s.db.WithContext(ctx).Model(&storedMessage{}).
		Select("channel, status, COUNT(*) as count").
		Where("tenant_id = ? AND created_at >= ? AND created_at <= ?", tenantID, since, until).
		Group("channel, status")
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]map[string]int64{}
	for _, r := range rows {
		if out[r.Channel] == nil {
			out[r.Channel] = map[string]int64{}
		}
		out[r.Channel][r.Status] = r.Count
	}
	return out, nil
}

// Close 释放资源。
func (s *SQLiteStore) Close() error { return nil }

func toStored(m *Message) *storedMessage {
	return &storedMessage{
		TenantID:   m.TenantID,
		RequestID:  m.RequestID,
		Channel:    m.Channel,
		Title:      m.Title,
		Content:    m.Content,
		Recipient:  m.Recipient,
		Status:     m.Status,
		Error:      m.Error,
		RetryCount: m.RetryCount,
		CostMs:     m.CostMs,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

func fromStored(r *storedMessage) *Message {
	return &Message{
		MessageID:  r.MessageID,
		TenantID:   r.TenantID,
		RequestID:  r.RequestID,
		Channel:    r.Channel,
		Title:      r.Title,
		Content:    r.Content,
		Recipient:  r.Recipient,
		Status:     r.Status,
		Error:      r.Error,
		RetryCount: r.RetryCount,
		CostMs:     r.CostMs,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}
