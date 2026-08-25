# AGENTS.md — internal/service

## 定位
业务逻辑层, 不碰 HTTP/路由。全部服务由 internal/app 容器构建并持有。
**新增服务 = 在 app.Build 中实例化并挂到 App 结构体。**

## 服务总览
| 服务 | 职责 | 依赖 |
|---|---|---|
| AuthService | 注册/登录/JWT/API Key/改密 | db, config |
| MessageService | 发送受理链路(幂等→限流→渲染→入队) | db, store.MessageStore, queue.Queue, RateLimiter |
| BatchService | 批量任务后台 runner | db, MessageService |
| PaymentService | 易支付下单/验签/回调/订阅生效 | db, SettingsService |
| SettingsService | DB 键值设置 | db |
| TemplateService | 模板 + 审核流 | db |
| RateLimiter(接口) | 内存/Redis 双实现 | — |

## 关键逻辑

### 幂等(MessageService.Send)
- 同租户 + 同 request_id 只受理一次: DB 唯一索引兜底 + Redis 缓存加速(快路径命中直接返回原 message_id, TTL 86400)。
- Redis 异常自动降级走 DB, 不阻断发送。
- 只有成功入队 ≥1 条才写幂等记录; 入队失败不写, 允许重试。

### 限流(RateLimiter 接口)
- 接口: `Allow(tenantID) / Type() / SetPerMinute(n)`。
- MemoryRateLimiter: 令牌桶, 单实例/降级用。
- RedisRateLimiter: 固定窗口, Lua `INCR+PEXPIRE` 原子计数, key `ratelimit:{tenant}`, 窗口 60s。
- Redis 异常放行(容错优先, 由审计兜底)。

### 批量任务(BatchService)
- Create 后 `go s.process(context.Background(), task.ID)`。
  **⚠️ 必须用 context.Background(): 请求 ctx 随 HTTP 结束即取消, 已踩坑。**
- 防并发: 内存 running map + DB `pending→running` 条件更新双重保险。
- 分片 100/片更新进度, 每片检查取消标记。

### 易支付签名(PaymentService)
- `SignParams`: 参数按 key 升序, 拼 `k=v&` 链, **跳过空值与 sign**, 末尾追加 `key=商户密钥`, 整体 MD5 小写。
- 回调链路: 验签 → pid 校验 → trade_status=TRADE_SUCCESS → 金额比对(±0.001) → 订单 paid → activateSubscription。
- 幂等: 已 paid 直接返回成功; 订阅同套餐未过期则顺延, 否则从现在起。

### 模板审核流(TemplateService)
- 创建模板自动生成 v1 待审版本(TemplateVersion), 模板 status=pending。
- **有待审版本时禁止再改**(UpdateTemplate 先 count pending, 非零即拒绝)。
- ApproveVersion: 版本 approved, 模板 content 更新为该版本内容, status→active。
- RejectVersion: 无其它 pending 时模板回退 rejected。

### 消息状态机
PENDING → SENDING → SUCCESS/FAILED/RETRYING/DEAD(定义见 store.Message)。
生命周期事件经 store.SaveEvent 落库(created|sending|success|failed|retry|dead)。

## 测试
- message_test.go: auto 渠道熔断降级(pickFallbackChannel, 全部熔断回退优先渠道)。
- ratelimit_test.go: Redis 固定窗口限流(窗口重置)、幂等 GET/SET Lua 脚本, 基于 miniredis。

## 约定
- EnqueueOne 是单发/批量共用的唯一入队入口: SaveMessage → SaveEvent → queue.Publish, 单条失败不中断整体。
- 业务错误用包内 errors.New(ErrRateLimited/ErrTemplateMissing/...), handler 层映射 HTTP 状态码。
- 渠道注册/熔断判定由容器注入函数(HasChannel/IsAvailable), 测试可替换。
- 模板渲染 `{{var}}` 正则替换, 缺失参数替换为空串。
