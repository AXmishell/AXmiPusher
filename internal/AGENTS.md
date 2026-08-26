# AGENTS.md

## 分层总览

- 组合根: internal/app。上帝对象, 装配一切。
- 全部包最终依赖 app, 禁止反向依赖。
- app 双身份: DI 容器 + 安装向导 HTTP 处理器(install.go)。
- 关键抽象走接口 + 双实现, 由配置切换:
  - service: 限流(内存/Redis)。
  - queue: 队列(inprocess/Kafka)。
  - store: 消息存储(SQLite/ClickHouse)。
  - channel: 熔断(内存/Redis), 渠道按 Sender 接口注册。
- api 与 worker 是两个进程, 共享同一个 app 容器。
- 本地模式: api 进程内嵌消费者(app.StartConsumer)。
- 生产模式: worker 进程独立跑消费者。

## 消息流转

- 发送: HTTP 受理 → service.MessageService → store.SaveMessage → queue.Publish。
- 消费: worker.Handle → registry.Dispatch → 渠道 Send。
- 状态回写: 发送中/成功/重试/死信, 全程写事件流(store.SaveEvent)。
- 限流在受理侧, 熔断在分发侧, 重试在消费侧。

## 包清单

| 包 | 职责 | 关键约束 |
|---|---|---|
| app | 组合根: 装配配置/DB/存储/队列/服务/渠道 | 上帝对象, 全包依赖它; 含安装向导路由; Reinit 不得重建同类型队列 |
| api | 路由装配 + 中间件 + HTTP 处理器(handler/) | 业务 API 需 RequireInstalled; 处理器只做薄层, 逻辑在 service |
| channel | 渠道适配层: Sender 接口 + Registry 分发 + 熔断 | 新渠道实现 Sender 并在 app.Build 注册; Dispatch 前必须 SetBreaker |
| compat | 兼容层: Server酱 v1/v2 接口 | 公开路由, 走兼容 key 鉴权 |
| config | 配置加载与持久化 | MP_* 环境变量 > config.yaml > 默认值; Save 由安装程序调用 |
| db | GORM 初始化 + AutoMigrate | 生产 PG / 本地 SQLite; SQLite 单写连接; migrations/ 为空 |
| models | 全部业务数据模型 | GORM 标签; 消息状态机见 store.Message; **无 Tenant 模型**(2026-08 已折叠入 User) |
| pkg | 通用小工具 | 仅 pkg/response: 统一 API 响应格式 |
| queue | 队列抽象 + 双实现 | inprocess/Kafka; Subscribe 阻塞; 消费逻辑在 worker |
| service | 业务逻辑层 | 后台任务必须用独立 ctx; 模板有待审核版本禁止修改 |
| store | 消息记录存储抽象 + 双实现 | SQLite/ClickHouse; 状态机 PENDING→SENDING→SUCCESS/FAILED/RETRYING/DEAD |
| worker | 消费者: 取任务 → 渠道分发 → 更新状态 | 同步重试 3 次; 熔断错误直接进死信, 不重试 |

## 死目录

- internal/install/ 为空, 全仓无引用。勿写入。

## CONVENTIONS

- 注释与日志一律中文。
- 配置优先级: MP_* 环境变量 > config.yaml > 默认值。
- 后台任务必须用独立 ctx(如 context.Background)。请求 ctx 在 HTTP 结束即取消。
- 不要重建同类型队列实例。消费者订阅旧实例后, 重建会导致消息无人消费。
- 模板存在待审核版本时禁止修改(service/template.go 会拒绝)。
- **租户已折叠入用户**(2026-08): 全部业务表 `tenant_id` 列名/参数名保留, 值 = 归属用户 ID; 限流/幂等/消息统计等按 tenantID 参数处一律传 user.ID。
