# AGENTS.md: internal/worker — 消息消费逻辑(被 api 内嵌 DBQueue 调用)

## 定位
消费者核心: 取 TaskMessage → 渠道分发 → 状态回写; 无独立进程(api 单进程内嵌)。

## 核心语义

### 惰性 getter(worker.go:22-27)
- `Worker` 构造只收两个闭包 `getStore`/`getRegistry`, 不含实例
- Reinit 重建 store/registry 后, Handle 每次调用时取最新实例

### Handle 无 error 返回(worker.go:35)
- 签名 `Handle(ctx, msg)` 不返回 error: 终态(SUCCESS/DEAD)全部内部落库, 吞掉发送结果
- 每次状态变更成对写 `UpdateStatus` + `SaveEvent`: SENDING(39-40) / RETRYING(46-50) / DEAD(71-74) / SUCCESS(77-81)

### 同步重试(worker.go:16-19, 44-67)
- `maxRetries=3`, 退避 `retryBackoff 2s × attempt`(1s/2s/3s)
- 重试前置 RETRYING + 写 retry 事件("第 %d 次重试")
- 前置 `select ctx.Done()`(52-54): 请求 ctx 取消立即 return, 不继续重试

### 熔断错误不重试(worker.go:61-65)
- `errors.As` 命中 `*channel.ErrCircuitOpen` 直接 break 走 DEAD(重试也快速失败, 无意义)

### 成功事件带耗时(worker.go:80)
- `Detail: fmt.Sprintf("耗时 %dms", cost)`, cost 自 Handle 开始计时(38 行)

## 已踩坑
- worker.go:22-23: store/registry 必须惰性 getter。构造时快照 → Reinit 后写进旧 store, 消息卡 PENDING — 已踩坑

## 与队列的关系
- 被 `app.StartConsumer`(app.go:183-189)的 `Queue.Subscribe` handler **每次调用 new**(185 行), 无单例缓存
- 队列层(DBQueue)只负责取消息/认领; 重试、状态机、死信判定全在 Handle 内, 队列不感知业务状态
- 消费侧不设熔断降级路由: 熔断已在 registry.Dispatch 内处理, Handle 只识别 ErrCircuitOpen
