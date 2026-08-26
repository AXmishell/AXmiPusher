package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// newTestStore 创建 :memory: SQLite 测试存储。
// 注意: :memory: 单连接 —— MaxOpenConns 必须为 1, 否则多连接各自持有独立内存库。
func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开 :memory: SQLite 失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层连接池失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	s, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteStore 失败: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return s
}

// insertMsg 插入一条消息并返回 MessageID。
func insertMsg(t *testing.T, s *SQLiteStore, m *Message) uint64 {
	t.Helper()
	id, err := s.SaveMessage(context.Background(), m)
	if err != nil {
		t.Fatalf("SaveMessage 失败: %v", err)
	}
	return id
}

// backdate 把消息 updated_at 回拨 duration, 模拟租约过期(崩溃残留)。
func backdate(t *testing.T, s *SQLiteStore, id uint64, duration time.Duration) {
	t.Helper()
	if err := s.db.Model(&storedMessage{}).
		Where("message_id = ?", id).
		Update("updated_at", time.Now().Add(-duration)).Error; err != nil {
		t.Fatalf("回拨 updated_at 失败: %v", err)
	}
}

func TestClaimPending(t *testing.T) {
	s := newTestStore(t)
	id1 := insertMsg(t, s, &Message{TenantID: 1, RequestID: "r1", Channel: "webhook", Title: "t1", Content: "c1", Recipient: "a@b.com", Status: "PENDING"})
	id2 := insertMsg(t, s, &Message{TenantID: 1, RequestID: "r2", Channel: "webhook", Title: "t2", Content: "c2", Recipient: "c@d.com", Status: "PENDING"})

	claimed, err := s.ClaimPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("ClaimPending 失败: %v", err)
	}
	if len(claimed) != 2 {
		t.Fatalf("第一次认领应返回 2 条, 实际 %d", len(claimed))
	}
	if claimed[0].MessageID != id1 || claimed[1].MessageID != id2 {
		t.Fatalf("认领应按 message_id 升序返回, 实际 [%d %d]", claimed[0].MessageID, claimed[1].MessageID)
	}
	// 认领返回全量载荷。
	if claimed[0].Content != "c1" || claimed[0].Recipient != "a@b.com" || claimed[1].RequestID != "r2" {
		t.Fatalf("认领应返回全量载荷, 实际 %+v", claimed)
	}

	// 状态应翻转为 SENDING, 且刷新 updated_at 作租约。
	got1, err := s.GetMessage(context.Background(), id1)
	if err != nil {
		t.Fatalf("GetMessage(id1) 失败: %v", err)
	}
	got2, err := s.GetMessage(context.Background(), id2)
	if err != nil {
		t.Fatalf("GetMessage(id2) 失败: %v", err)
	}
	if got1.Status != "SENDING" || got2.Status != "SENDING" {
		t.Fatalf("认领后状态应为 SENDING, 实际 %q %q", got1.Status, got2.Status)
	}
	if got1.UpdatedAt.Before(got1.CreatedAt) {
		t.Fatalf("认领应刷新 updated_at 作租约: created=%v updated=%v", got1.CreatedAt, got1.UpdatedAt)
	}

	// 第二次认领应返回 0 条(不重复认领)。
	again, err := s.ClaimPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("第二次 ClaimPending 失败: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("已认领消息不应被重复认领, 第二次返回 %d 条", len(again))
	}
}

func TestReapStale(t *testing.T) {
	s := newTestStore(t)
	id1 := insertMsg(t, s, &Message{TenantID: 1, RequestID: "r1", Channel: "webhook", Title: "t1", Content: "c1", Recipient: "a@b.com", Status: "SENDING", RetryCount: 0})
	backdate(t, s, id1, 2*time.Minute)

	n, err := s.ReapStale(context.Background(), time.Minute, 5)
	if err != nil {
		t.Fatalf("ReapStale 失败: %v", err)
	}
	if n != 1 {
		t.Fatalf("超租约 SENDING 应回收 1 条, 实际 %d", n)
	}
	m1, err := s.GetMessage(context.Background(), id1)
	if err != nil {
		t.Fatalf("GetMessage 失败: %v", err)
	}
	if m1.Status != "PENDING" {
		t.Fatalf("回拨 2 分钟超 1 分钟租约, 应复位 PENDING, 实际 %q", m1.Status)
	}
	if m1.RetryCount != 1 {
		t.Fatalf("回收后 retry_count 应 +1 为 1, 实际 %d", m1.RetryCount)
	}

	// 重复"认领→回拨→回收"至上限: retry_count 1→5, 第 5 次 retry_count+1=5 >= maxAttempts 置 DEAD。
	// 每次回收把消息复位 PENDING, 需重新认领回 SENDING 才会再被回收(真实崩溃恢复闭环)。
	for i := 0; i < 4; i++ {
		claimed, err := s.ClaimPending(context.Background(), 10)
		if err != nil {
			t.Fatalf("循环第 %d 次认领失败: %v", i, err)
		}
		if len(claimed) != 1 {
			t.Fatalf("循环第 %d 次应能认领 1 条(复位后重新入队), 实际 %d", i, len(claimed))
		}
		backdate(t, s, id1, 2*time.Minute)
		if _, err := s.ReapStale(context.Background(), time.Minute, 5); err != nil {
			t.Fatalf("循环第 %d 次 ReapStale 失败: %v", i, err)
		}
	}
	m1, err = s.GetMessage(context.Background(), id1)
	if err != nil {
		t.Fatalf("GetMessage 失败: %v", err)
	}
	if m1.Status != "DEAD" {
		t.Fatalf("达到回收上限应置 DEAD, 实际 %q (retry_count=%d)", m1.Status, m1.RetryCount)
	}
	if m1.RetryCount != 5 {
		t.Fatalf("DEAD 时 retry_count 应为 5, 实际 %d", m1.RetryCount)
	}
	if m1.Error != "认领超限(数据库队列)" {
		t.Fatalf("DEAD 错误信息应为认领超限, 实际 %q", m1.Error)
	}

	// RETRYING 同样被回收。
	id2 := insertMsg(t, s, &Message{TenantID: 1, RequestID: "r2", Channel: "webhook", Title: "t2", Content: "c2", Recipient: "c@d.com", Status: "RETRYING", RetryCount: 0})
	backdate(t, s, id2, 2*time.Minute)
	n, err = s.ReapStale(context.Background(), time.Minute, 5)
	if err != nil {
		t.Fatalf("ReapStale(RETRYING) 失败: %v", err)
	}
	if n != 1 {
		t.Fatalf("超租约 RETRYING 应同样回收 1 条, 实际 %d", n)
	}
	m2, err := s.GetMessage(context.Background(), id2)
	if err != nil {
		t.Fatalf("GetMessage 失败: %v", err)
	}
	if m2.Status != "PENDING" || m2.RetryCount != 1 {
		t.Fatalf("RETRYING 回收应复位 PENDING 且 retry_count=1, 实际 %q/%d", m2.Status, m2.RetryCount)
	}
}

func TestReapFresh(t *testing.T) {
	s := newTestStore(t)
	id := insertMsg(t, s, &Message{TenantID: 1, RequestID: "r1", Channel: "webhook", Title: "t1", Content: "c1", Recipient: "a@b.com", Status: "SENDING", RetryCount: 0})

	// updated_at = 插入时刻(租约未过期)。
	n, err := s.ReapStale(context.Background(), time.Minute, 5)
	if err != nil {
		t.Fatalf("ReapStale 失败: %v", err)
	}
	if n != 0 {
		t.Fatalf("租约未过期不应回收, 实际回收 %d 条", n)
	}
	m, err := s.GetMessage(context.Background(), id)
	if err != nil {
		t.Fatalf("GetMessage 失败: %v", err)
	}
	if m.Status != "SENDING" {
		t.Fatalf("租约未过期消息不应被改动, 实际 %q", m.Status)
	}
	if m.RetryCount != 0 {
		t.Fatalf("租约未过期消息 retry_count 不应变, 实际 %d", m.RetryCount)
	}
}

func TestReapClaimExclusive(t *testing.T) {
	s := newTestStore(t)
	id := insertMsg(t, s, &Message{TenantID: 1, RequestID: "r1", Channel: "webhook", Title: "t1", Content: "c1", Recipient: "a@b.com", Status: "PENDING", RetryCount: 0})

	claimed, err := s.ClaimPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("ClaimPending 失败: %v", err)
	}
	if len(claimed) != 1 || claimed[0].MessageID != id {
		t.Fatalf("应认领 1 条消息, 实际 %d 条", len(claimed))
	}

	// 认领后立刻回拨 updated_at, 模拟消费者崩溃残留的过期租约。
	backdate(t, s, id, 2*time.Minute)
	n, err := s.ReapStale(context.Background(), time.Minute, 5)
	if err != nil {
		t.Fatalf("ReapStale 失败: %v", err)
	}
	if n != 1 {
		t.Fatalf("过期租约应回收 1 条, 实际 %d", n)
	}
	m, err := s.GetMessage(context.Background(), id)
	if err != nil {
		t.Fatalf("GetMessage 失败: %v", err)
	}
	if m.Status != "PENDING" {
		t.Fatalf("回收后应复位 PENDING, 实际 %q", m.Status)
	}

	// 再次认领可领: 崩溃恢复闭环。
	again, err := s.ClaimPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("二次 ClaimPending 失败: %v", err)
	}
	if len(again) != 1 || again[0].MessageID != id {
		t.Fatalf("回收后应可再次认领, 实际 %d 条", len(again))
	}
	m, err = s.GetMessage(context.Background(), id)
	if err != nil {
		t.Fatalf("GetMessage 失败: %v", err)
	}
	if m.Status != "SENDING" {
		t.Fatalf("二次认领应置回 SENDING, 实际 %q", m.Status)
	}
	if m.RetryCount != 1 {
		t.Fatalf("二次认领后 retry_count 应保持回收时值 1(认领不改变 retry_count), 实际 %d", m.RetryCount)
	}
}

// fakePGDialector 仅报告方言名为 postgres 的假 dialector: 让 GORM 按 postgres 方言构建 SQL,
// 但不建立任何真实连接(真实 postgres.New 会经 pgx 立即拨号, 无 PG 环境时测试无法进行)。
type fakePGDialector struct{}

func (fakePGDialector) Name() string { return "postgres" }
func (fakePGDialector) Initialize(db *gorm.DB) error {
	// 真实 dialector 在 Initialize 里注册默认回调, 假实现需照做, 否则查询处理器为空、SQL 不生成。
	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{})
	return nil
}
func (fakePGDialector) Migrator(*gorm.DB) gorm.Migrator                { return nil }
func (fakePGDialector) DataTypeOf(*schema.Field) string                { return "text" }
func (fakePGDialector) DefaultValueOf(*schema.Field) clause.Expression { return nil }
func (fakePGDialector) BindVarTo(writer clause.Writer, _ *gorm.Statement, _ interface{}) {
	writer.WriteByte('?')
}
func (fakePGDialector) QuoteTo(clause.Writer, string)               {}
func (fakePGDialector) Explain(sql string, _ ...interface{}) string { return sql }

func TestPGClaimShape(t *testing.T) {
	// 不连真 PG: 假 postgres 方言 + DryRun 会话生成认领 SQL, 只验证形状。
	pg, err := gorm.Open(fakePGDialector{}, &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("打开 PG DryRun 会话失败: %v", err)
	}
	s := &SQLiteStore{db: pg}
	var recs []storedMessage
	stmt := claimQuery(s.db, 10).Find(&recs).Statement
	sqlStr := stmt.SQL.String()
	if !strings.Contains(sqlStr, "FOR UPDATE SKIP LOCKED") {
		t.Fatalf("PG 认领 SQL 应含 FOR UPDATE SKIP LOCKED, 实际: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, "ORDER BY message_id") {
		t.Fatalf("PG 认领 SQL 应含 ORDER BY message_id, 实际: %s", sqlStr)
	}
}

// fakeMySQLDialector 仅报告方言名为 mysql 的假 dialector(同 fakePGDialector 思路)。
type fakeMySQLDialector struct{}

func (fakeMySQLDialector) Name() string { return "mysql" }
func (fakeMySQLDialector) Initialize(db *gorm.DB) error {
	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{})
	return nil
}
func (fakeMySQLDialector) Migrator(*gorm.DB) gorm.Migrator                { return nil }
func (fakeMySQLDialector) DataTypeOf(*schema.Field) string                { return "text" }
func (fakeMySQLDialector) DefaultValueOf(*schema.Field) clause.Expression { return nil }
func (fakeMySQLDialector) BindVarTo(writer clause.Writer, _ *gorm.Statement, _ interface{}) {
	writer.WriteByte('?')
}
func (fakeMySQLDialector) QuoteTo(clause.Writer, string)               {}
func (fakeMySQLDialector) Explain(sql string, _ ...interface{}) string { return sql }

func TestMySQLClaimShape(t *testing.T) {
	// MySQL 5.7 不支持 SKIP LOCKED(8.0 起才有): 认领 SQL 应含 FOR UPDATE 但绝不含 SKIP LOCKED。
	my, err := gorm.Open(fakeMySQLDialector{}, &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("打开 MySQL DryRun 会话失败: %v", err)
	}
	s := &SQLiteStore{db: my}
	var recs []storedMessage
	stmt := claimQuery(s.db, 10).Find(&recs).Statement
	sqlStr := stmt.SQL.String()
	if !strings.Contains(sqlStr, "FOR UPDATE") {
		t.Fatalf("MySQL 认领 SQL 应含 FOR UPDATE, 实际: %s", sqlStr)
	}
	if strings.Contains(sqlStr, "SKIP LOCKED") {
		t.Fatalf("MySQL 5.7 认领 SQL 不应含 SKIP LOCKED, 实际: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, "ORDER BY message_id") {
		t.Fatalf("MySQL 认领 SQL 应含 ORDER BY message_id, 实际: %s", sqlStr)
	}
}
