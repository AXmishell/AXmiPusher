# PROJECT KNOWLEDGE BASE — messagepusher

**Generated:** 2026-08-26 00:59
**Commit:** 23bbccb
**Branch:** deploy

## OVERVIEW
消息推送平台: 统一受理消息 → 队列 → 多渠道发送(webhook/email/apns/fcm/站内信) → 回执统计。Go 后端(单仓库双进程) + 两个独立 React 前端, 支持本地模式单机运行与生产分布式部署。

## STRUCTURE
```
messagepusher/
├── cmd/             # 3 入口: api(HTTP+本地内嵌消费者) / worker(独立消费者) / redis-mock(开发工具)
├── internal/        # 全部后端: app(组合根) + api/channel/compat/config/db/models/pkg/queue/service/store/worker
├── web/
│   ├── user/        # 用户中心 (Vite+React+AntD Pro, :5173)
│   └── admin/       # 管理后台 (同上, :5174, 随机路径 base)
├── deploy/          # Docker compose + 部署脚本 + context/(产物提交)
├── scripts/         # 本地工具: hook-receiver.ps1(:9090) / mock-smtp.ps1(:2525)
├── migrations/      # 空目录(建表走 GORM AutoMigrate, 无版本化迁移)
└── openapi.yaml     # API 规范
```

## WHERE TO LOOK
| 任务 | 位置 | 备注 |
|------|------|------|
| 消息受理链路 | internal/service/message.go | 幂等→限流→模板→入队 |
| 渠道/熔断 | internal/channel/ | 5 渠道 + Breaker(内存/Redis) |
| 路由/响应 | internal/api/ + internal/pkg/response | 统一 {code,message,data} |
| 安装向导 | internal/app/install.go + install.html | 由 app 包承载(非 api) |
| 支付/订阅 | internal/service/payment.go | 易支付 MD5 签名 |
| 前端页面 | web/{user,admin}/src/pages/ | ProTable 模式 |
| 部署编排 | deploy/context/ | 单机 compose + 产物 |

## CODE MAP
| Symbol | Type | Location | Role |
|--------|------|----------|------|
| App | struct | internal/app/app.go | 组合根/DI 容器, 全仓库依赖中心 |
| MessageService.Send | method | internal/service/message.go | 发送受理核心(幂等/限流/模板) |
| Registry.Dispatch | method | internal/channel/channel.go | 渠道分发+熔断集成 |
| Breaker(2 实现) | interface | internal/channel/breaker*.go | 熔断状态机 |
| Queue(2 实现) | interface | internal/queue/ | inprocess/kafka |
| MessageStore(2 实现) | interface | internal/store/ | sqlite/clickhouse |
| NewRouter | func | internal/api/router.go | 全路由装配 |
| EnqueueOne | method | internal/service/message.go | 单发/批量共用入队 |

## CONVENTIONS
- **中文注释与日志**(代码与文档均中文, 勿改英文)
- 配置优先级: `MP_*` 环境变量 > config.yaml > 默认值 (internal/config)
- 接口+双实现模式: Queue/Store/RateLimiter/Breaker 均内存(本地)与 Redis/Kafka/ClickHouse(生产)可插拔
- 鉴权: 网页端 JWT(Authorization Bearer), 服务端 API Key(mp_ 前缀, SHA-256 哈希存库); 中间件用惰性 getAuth 闭包(容器 Reinit 后取最新实例)
- 管理后台随机路径: 构建期 admin base 必须等于运行期 `MP_ADMIN_PATH`(默认约定 b322aa9602150d0c)
- 列表接口对齐 AntD Pro 分页: `current/pageSize` 请求, `{data,total,success}` 响应
- 幂等: request_id + DB 唯一索引兜底 + Redis SETNX 加速
- 部署产物(deploy/context/api + web dist)提交入库; 凭据/运行态绝不提交

## ANTI-PATTERNS (THIS PROJECT)
- **绝不提交**: *.key/*.pem/*.p8, deploy/server.local.json, deploy/*.local*, config.yaml(含 JWT 密钥), install.lock, data/, *.log
- 后台 goroutine 不得用请求 ctx(HTTP 结束即取消 → 消息丢) — 必须 context.Background()
- 不要重建同类型队列实例(消费者订阅旧实例, 重建后消息无人消费) — 已踩坑
- 模板存在"待审核版本"时禁止修改内容
- Gin 根级 catch-all(`/*filepath`)与 /api 路由冲突 panic — 前端托管用 NoRoute 兜底
- smtpSend 必须设连接 deadline(非 SMTP 端口会永久阻塞 worker) — 已踩坑
- Vite 代理只写 `/api/v1`, 勿扩成 `/api`(误伤 /api-keys 等 SPA 路由)
- 服务器/容器内禁止 go/node 构建(1G 内存 OOM) — 产物一律本地构建

## UNIQUE STYLES
- 响应体: `{code:0,message:"ok",data}`; 业务码 0/40000/40100/40300/40400/40900/42900/50000; 消息发送受理返回 HTTP 202
- 消息状态机: PENDING→SENDING→SUCCESS/FAILED/RETRYING/DEAD/CANCELLED; 熔断错误不重试直接 DEAD
- 安装向导: install.lock 门控, 未安装时业务 API 返回 503, 引导 /install
- 兼容层: Server酱 v1(/api/sc/{key}.send) + v2(/api/sctapi/{key}.send), 返回格式照抄原版

## COMMANDS
```bash
# 后端(本地模式: SQLite + 进程内队列, 内嵌消费者)
go run ./cmd/api                  # :8080
go run ./cmd/redis-mock           # 模拟 Redis :16379(可选)

# 前端
cd web/user && npm run dev        # :5173
cd web/admin && npm run dev       # :5174

# 测试
go test ./...

# 构建部署产物(本地, 供 git 提交)
powershell -File deploy/build-artifacts.ps1 -AdminPath b322aa9602150d0c

# 单机 Docker 编排部署
powershell -File deploy/docker-deploy.ps1

# 云端 git 部署流
git push cloud deploy
ssh mpcloud "sudo bash /opt/messagepusher-src/deploy/cloud-deploy.sh"

# 本地测试工具
powershell -File scripts/mock-smtp.ps1    # :2525 → data/smtp.log
powershell -File scripts/hook-receiver.ps1 # :9090 → data/hook.log
```

## NOTES
- Go 1.25(go.mod) vs Dockerfile golang:1.26 存在版本偏差
- 两个前端 package.json 的 name 均为 "messagepusher-user"(复制粘贴遗留)
- handler/rand.go 命名误导: 实为共享助手(CurrentUser/randomHex)
- internal/install/ 为空死目录; migrations/ 空(生产 PG 建议补版本化迁移)
- 进程内队列在进程重启时丢弃在途消息(生产切 Kafka 消除)
- git remote: `cloud`(云 bare repo) + `origin`(Gitee 备份); ~/.ssh/config 有 `mpcloud` 别名
- 云部署: Debian12 1C/1G, 单机 compose(api+redis), 数据在 /opt/messagepusher-docker/appdata
