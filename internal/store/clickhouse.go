package store

import (
	"context"
	"fmt"
	"time"
)

// ClickHouseStore 基于 ClickHouse 的消息存储(生产模式)。
// M1 阶段为骨架实现, 待 ClickHouse 环境就绪后完善建表与查询。
type ClickHouseStore struct {
	dsn string
}

// NewClickHouseStore 创建 ClickHouse 消息存储。
func NewClickHouseStore(dsn string) (*ClickHouseStore, error) {
	if dsn == "" {
		return nil, fmt.Errorf("ClickHouse DSN 为空")
	}
	return &ClickHouseStore{dsn: dsn}, nil
}

// SaveMessage 创建消息记录。
func (s *ClickHouseStore) SaveMessage(ctx context.Context, m *Message) (uint64, error) {
	return 0, fmt.Errorf("ClickHouseStore: 未实现(待 ClickHouse 环境就绪)")
}

// UpdateStatus 更新消息状态。
func (s *ClickHouseStore) UpdateStatus(ctx context.Context, messageID uint64, status, errMsg string) error {
	return fmt.Errorf("ClickHouseStore: 未实现(待 ClickHouse 环境就绪)")
}

// GetMessage 查询单条消息。
func (s *ClickHouseStore) GetMessage(ctx context.Context, messageID uint64) (*Message, error) {
	return nil, fmt.Errorf("ClickHouseStore: 未实现(待 ClickHouse 环境就绪)")
}

// QueryMessages 分页查询。
func (s *ClickHouseStore) QueryMessages(ctx context.Context, tenantID uint64, f MessageFilter, page, size int) ([]*Message, int64, error) {
	return nil, 0, fmt.Errorf("ClickHouseStore: 未实现(待 ClickHouse 环境就绪)")
}

// SaveEvent 记录消息事件。
func (s *ClickHouseStore) SaveEvent(ctx context.Context, e *MessageEvent) error {
	return fmt.Errorf("ClickHouseStore: 未实现(待 ClickHouse 环境就绪)")
}

// StatsByStatus 按状态统计。
func (s *ClickHouseStore) StatsByStatus(ctx context.Context, tenantID uint64, since, until time.Time) (map[string]int64, error) {
	return nil, fmt.Errorf("ClickHouseStore: 未实现(待 ClickHouse 环境就绪)")
}

// StatsByChannel 按渠道+状态统计。
func (s *ClickHouseStore) StatsByChannel(ctx context.Context, tenantID uint64, since, until time.Time) (map[string]map[string]int64, error) {
	return nil, fmt.Errorf("ClickHouseStore: 未实现(待 ClickHouse 环境就绪)")
}

// Close 释放资源。
func (s *ClickHouseStore) Close() error { return nil }
