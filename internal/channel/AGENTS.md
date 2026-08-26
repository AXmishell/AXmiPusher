# AGENTS.md: internal/channel 渠道适配层

## 文件地图
- channel.go: Sender 接口 + Registry 分发, 熔断集成入口
- webhook.go / email.go / apns.go / fcm.go / inapp.go: 五个渠道
- smtp.go: 低层 SMTP 助手(smtpSend / buildMessage), 不含渠道逻辑
- breaker.go: 内存熔断器 CircuitBreaker
- breaker_redis.go: Redis 熔断器(多实例共享), Lua 原子脚本
- breaker_test.go / breaker_redis_test.go / channel_test.go: 熔断状态机 + 签名/MIME 单元测试

## 核心模式
- Sender 接口: Type() 返回渠道名, Send(ctx, msg) 发消息。所有渠道实现它。
- Registry 分发: NewRegistry → Register(sender) → Dispatch。Dispatch 先查 senders 表, 再走熔断。
- 熔断流程: Dispatch 内 Allow → Send → 成功调 Success / 失败调 Failure。ErrCircuitOpen 即熔断, 上层不重试。
- IsAvailable(tenantID, channel): 已注册且未熔断, 供 auto 路由做熔断降级。
- Breaker 双实现: 内存 CircuitBreaker(单机) + RedisCircuitBreaker(多实例)。状态 0 闭合 / 1 熔断 / 2 半开, 按 (归属用户, 渠道) 维度隔离。
- Redis 熔断: 状态存 breaker:{用户ID}:{channel} HASH, 全部状态转移走 Lua 原子脚本, 保证多实例并发正确。
- 新增渠道 = 实现 Sender + NewXxxSender 构造 + 在组装处 Register。无插件机制。

## 配置优先级
- 用户自定义: models.Channel 表, 按 tenant_id(=归属用户 ID) + type + status=active 查, Config 字段 JSON 解析成各渠道 Config。
- email 特例: 用户配置缺失时降级读平台 Settings("smtp" key)。apns / fcm 只认用户配置。
- 结论: 用户 Channel(type 维度, 用户覆盖) > 平台 Settings(SMTP)。

## 已踩坑(必读)
- smtpSend 必须设连接 deadline: 撞上非 SMTP 端口(只收不发 banner 的服务)会永久阻塞 worker。连接建立后立即 conn.SetDeadline(now+15s), 会话总超时。
- 端口 465 隐式 TLS, 587/25 走 STARTTLS(若支持), 25 常被反垃圾策略拦截。
- 熔断默认: 阈值 3 次连续失败 → OPEN, 冷却 30s → HALF_OPEN 放行一个探针; 探针成功回 CLOSED, 失败立即重新 OPEN。
- worker 对熔断错误不重试直接 DEAD: ErrCircuitOpen 是熔断错误, 不是渠道错误, 重试无意义。
- webhook 签名: X-MP-Signature = HMAC-SHA256(sub.Secret, body) hex, 业务方验签防伪造。
- 各渠道 Send 自行校验空 Recipient, 分别报错。

## 测试约定
- Redis 熔断测试用 miniredis.RunT + redis.Client, 但冷却判断走 Go 墙钟(time.Now()), Lua 里 now 来自 ARGV。
- 不要用 miniredis.FastForward: 只影响 miniredis 内部时钟, 对 Lua 中的 Go 墙钟无效果。
- 用真实小冷却(如 20ms) + time.Sleep 等冷却结束。内存熔断测试同款: 小冷却 + sleep。
- channel_test.go 为纯单元测试, 不连网: APNs ES256 JWT 签名可验证, FCM RS256 key 解析, 邮件 MIME 头与 base64 正文。

## 渠道速查
- webhook: POST 租户回调地址, 任一订阅成功即整体成功。
- email: smtpSend, 租户配置优先, 平台默认兜底。
- apns: HTTP/2 + ES256 JWT 令牌认证; 400/403/404/410 视为设备 token 失效, 不重试。
- fcm: 服务账号 RS256 JWT 换 OAuth2 access token, 再调 messages:send。
- inapp: 写 InappMessage 表; Recipient 为邮箱或 "all"(该归属用户的全部收件人)。
