# AGENTS.md: internal/store — 消息存储(与业务库同库)

## 定位
- MessageStore 接口 + 唯一实现 SQLiteStore(名称沿用旧版, 实为 GORM store, SQLite/PG 通用; 消息与业务库同库, 消息写库即入队)。

## 文件
- store.go: 接口(9 方法: Save/Update/Get/Query/SaveEvent/Stats×2/Claim/Reap) + 模型(Message/MessageEvent/MessageFilter) + 状态机注释
- sqlite.go: GORM 实现(私有 storedMessage 模型 + 全部方法)
- sqlite_test.go: 认领/租约回收单测

## 核心语义
- **消息状态机**: PENDING→SENDING→SUCCESS/FAILED/RETRYING/DEAD(store.go:10, 61-64); 状态常量实值在 internal/models(MsgPending 等), 本包内直接写字符串字面量
- **认领 ClaimPending**(sqlite.go:178-217): 方言分支 — postgres 用 SELECT ... FOR UPDATE SKIP LOCKED 行锁 + 逐条 UPDATE(事务内, 锁持有到提交, 防并发重复认领); sqlite 单写连接(db 包 SetMaxOpenConns(1))天然串行, 无需锁
- **回收 ReapStale**(sqlite.go:219-236): 单条 UPDATE ... CASE WHEN 原子完成 — retry_count+1, 超 maxAttempts 置 DEAD("认领超限(数据库队列)")否则复位 PENDING; updated_at 过期判断用 GORM 参数绑定传 time.Time(与写入格式一致), 不做字符串拼接
- **索引契约**: (status, updated_at) 联合索引 idx_status_updated 专供轮询/租约回收(sqlite.go:21, 26); idx_tenant_created 供租户分页; request_id/channel/recipient 各带单列索引
- **事件流**: message_events 表记录 created|sending|success|failed|retry|dead(SaveEvent, store.go:30)
- **统计**: StatsByStatus / StatsByChannel(store.go:57-60)GROUP BY 聚合, 走 created_at 时间范围; 消息表与事件表分离, 事件表只追加不更新
- **ID**: MessageID 自增主键, SaveMessage 写库后回填 m.MessageID(sqlite.go:63)供后续状态回写与幂等使用

## 已踩坑/必须
- sqlite.go:220: updated_at 比较必须 GORM 参数绑定, 不做字符串拼接(格式不一致 → 租约永不回收)
- 分页 size 钳制: <1 || >100 回退 20, page <1 回退 1(sqlite.go:84-90)
- GORM 模型私有 storedMessage, 经 toStored/fromStored 显式转换(sqlite.go:241-274), 勿直接改字段结构
- Close 为 no-op(sqlite.go:239): 连接归 db 包管理, 勿在此关库
- 认领按 message_id ASC 取最早入队, 保证 FIFO
- 查询排序实按 message_id DESC(非 created_at): 注释"按创建时间倒序"与实际实现不一致, 改排序时留意
- UpdateStatus 用 map Updates 整体覆盖 status/error/updated_at(sqlite.go:68-72), 勿改成 struct 更新(零值字段会被跳过)

## 与消息队列的关系
- ClaimPending/ReapStale 由 internal/queue DBQueue 调用, 是本项目"数据库轮询队列"的存储底座; 队列无独立中间件, 消息表即 outbox
