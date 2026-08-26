package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	Status     string    `gorm:"size:16;not null;index:idx_messages_status;index:idx_status_updated,priority:1" json:"status"`
	Error      string    `gorm:"size:1024" json:"error"`
	RetryCount int       `gorm:"not null;default:0" json:"retry_count"`
	CostMs     int64     `gorm:"not null;default:0" json:"cost_ms"`
	CreatedAt  time.Time `gorm:"index:idx_tenant_created" json:"created_at"`
	UpdatedAt  time.Time `gorm:"index:idx_status_updated,priority:2" json:"updated_at"`
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

// claimQuery 构造认领查询: postgres 分支附加 FOR UPDATE SKIP LOCKED 行锁,
// 保证多 worker 并发认领互斥(跳过已被锁定的行)。由 ClaimPending 与 PG 方言 SQL 形状测试共用。
func claimQuery(db *gorm.DB, limit int) *gorm.DB {
	q := db.Model(&storedMessage{}).
		Where("status = 'PENDING'").
		Order("message_id ASC").
		Limit(limit)
	if db.Dialector.Name() == "postgres" {
		q = q.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
	}
	return q
}

// ClaimPending 认领最多 limit 条 PENDING 消息(置为 SENDING 并刷新 updated_at 作租约), 返回被认领消息(含全量载荷)。
// 事务内 SELECT + 逐条 UPDATE: postgres 行锁持有到提交, 防止并发重复认领; sqlite 单写连接天然串行。
func (s *SQLiteStore) ClaimPending(ctx context.Context, limit int) ([]*Message, error) {
	var recs []storedMessage
	now := time.Now()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := claimQuery(tx, limit).Find(&recs).Error; err != nil {
			return err
		}
		for i := range recs {
			if err := tx.Model(&storedMessage{}).
				Where("message_id = ?", recs[i].MessageID).
				Updates(map[string]any{"status": "SENDING", "updated_at": now}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]*Message, 0, len(recs))
	for i := range recs {
		out = append(out, fromStored(&recs[i]))
	}
	return out, nil
}

// ReapStale 回收租约超时的 SENDING/RETRYING 消息: retry_count+1; 若 retry_count+1 >= maxAttempts 置 DEAD, 否则复位 PENDING。
// updated_at 过期判断用 GORM 参数绑定传 time.Time(与写入格式一致), 不做字符串拼接。
func (s *SQLiteStore) ReapStale(ctx context.Context, lease time.Duration, maxAttempts int) (int64, error) {
	cutoff := time.Now().Add(-lease)
	res := s.db.WithContext(ctx).Exec(
		`UPDATE messages SET
			status = CASE WHEN retry_count+1 >= ? THEN 'DEAD' ELSE 'PENDING' END,
			error = CASE WHEN retry_count+1 >= ? THEN '认领超限(数据库队列)' ELSE error END,
			retry_count = retry_count + 1,
			updated_at = ?
		WHERE status IN ('SENDING','RETRYING') AND updated_at < ?`,
		maxAttempts, maxAttempts, time.Now(), cutoff,
	)
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
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
