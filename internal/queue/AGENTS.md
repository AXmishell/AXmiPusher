# AGENTS.md: internal/queue — 数据库轮询队列

## 定位
Queue 接口抽象, 唯一实现 DBQueue: messages 表即 outbox, 消息写库即入队, 无独立队列中间件。

## 文件
- queue.go: Queue 接口 + TaskMessage + Handler
- dbqueue.go: DBQueue 轮询实现
- dbqueue_test.go: 单测

## 核心语义
- **Publish 恒 no-op**(dbqueue.go:59-64): 入队已在 service.EnqueueOne → store.SaveMessage 完成, 轮询按 status=PENDING 认领
- **轮询循环 round 顺序固定**: ① ReapStale 回收 → ② 认领数量 = min(batch, 空闲槽位数)(dbqueue.go:114-123, 认领即交 worker 消除批量堆积) → ③ 交 worker 调 handler; 无空闲槽位跳过本轮(117-119)
- **租约**: ClaimPending 刷新 updated_at 作租约; 超时 SENDING/RETRYING 由 store.ReapStale 复位 PENDING(retry_count++)或超限置 DEAD — 崩溃/取消恢复路径
- **handler 错误分支是防御性的**(dbqueue.go:161-165): 生产链路 worker.Handle 内部已处理终态且不返回 error
- **Close 幂等**(closeOnce + wg.Wait, dbqueue.go:168-173), 等待全部 worker 结束

## 已踩坑
dbqueue.go:17/29-31 — getStore 必须传惰性函数而非 store 实例: 容器 Reinit 重建 store 后仍取最新实例; 构造时快照 → 切库后轮询旧库, 消息静默丢失。

## 约定
- 轮询参数默认值在构造内兜底(500ms / 100 / 4 / 5 / 300s, dbqueue.go:33-47)
- ReapStale/ClaimPending 失败仅 log 不中断本轮; 用 `ctx.Err() == nil` 抑制关闭噪音
- TaskMessage 载荷自 store.Message 逐字段复制(dbqueue.go:147-160)
- Subscribe 阻塞直到 ctx 取消或 Close

## 与其他包关系
- 依赖 store: ClaimPending 认领 / ReapStale 回收
- 被 app.StartConsumer 调用(Reinit 保留 DBQueue 实例, getStore 闭包惰性取重建后的 store)
- 消费逻辑在 internal/worker: 本包只负责取消息, 不感知发送/渠道/状态机
