# PROJECT KNOWLEDGE BASE — messagepusher

**Generated:** 2026-08-26 16:40
**Commit:** ac065b8
**Branch:** deploy

## OVERVIEW
消息推送平台: 统一受理消息 → 队列 → 多渠道发送(webhook/email/apns/fcm/站内信) → 回执统计。Go 后端(单仓库双进程) + 两个独立 React 前端, 支持本地模式单机运行与生产分布式部署。

## STRUCTURE
```
messagepusher/
├── cmd/             # 4 入口: api(HTTP+本地内嵌消费者) / worker(独立消费者) / web(前端托管双端口) / redis-mock(开发工具)
├── internal/        # 全部后端: app(组合根) + api/channel/compat/config/db/models/pkg/queue/service/store/worker
├── web/
│   ├── user/        # 用户中心 (Vite+React+AntD Pro, :19876 生产 / :5173 dev)
│   └── admin/       # 管理后台 (同上, :19877 生产 / :5174 dev, 随机路径 base)
├── deploy/          # 安装分发(install.sh/pack-install.ps1) + 云端编译部署脚本 + context/(安装包产物)
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
| 前端托管 | cmd/web/ | 独立双端口 19876/19877, 读 config.yaml |
| 安装分发 | deploy/install.sh + pack-install.ps1 | 详见 deploy/AGENTS.md |
| 部署编排 | deploy/cloud-build-deploy.sh | 云端编译流(git push → 云端 go/npm build) |

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
- 鉴权(两套独立体系, JWT `kind` 双向隔离): 用户中心 = users 表, JWT kind=user; 管理后台 = admins 表(超管/普通管理员), JWT kind=admin, 管理员 token 进不了用户端点、用户 token 进不了 /admin; 服务端 API Key(mp_ 前缀, SHA-256 哈希存库); 中间件用惰性 getAuth 闭包(容器 Reinit 后取最新实例)
- **租户已折叠入用户**(2026-08): 无 tenants 表, User 直接携带 name/quota/plan_id; 全部业务表 `tenant_id` 列名保留, 值 = 归属用户 ID
- 管理后台随机路径: **admin 前端构建用相对 base('./'), 运行时任意前缀可用且支持轮换**(安装向导从 dist index.html 解析 base 兜底); 旧约定"构建 base 必须=MP_ADMIN_PATH"已废弃
- 配置契约: `config.yaml` 路径硬编码 CWD(无 env 覆盖); 25 个 `MP_*` env(MP_ENV/MP_DB_*/MP_QUEUE_TYPE/MP_KAFKA_*/MP_STORE_TYPE/MP_CH_DSN/MP_JWT_SECRET/MP_ADMIN_PATH/MP_REDIS_*/MP_USER_PORT/MP_ADMIN_PORT/MP_API_TARGET/MP_PORT); config.yaml 由安装向导生成, 缺失=未安装(503+/install)
- 列表接口对齐 AntD Pro 分页: `current/pageSize` 请求, `{data,total,success}` 响应
- 幂等: request_id + DB 唯一索引兜底 + Redis SETNX 加速
- **端口固化**: 主程序 8080 / 用户中心 19876 / 管理后台 19877, 由安装向导写入 config.yaml(web.user_port/admin_port/api_target); cmd/web 读配置, 优先级 MP_* env > config.yaml > 默认
- 密码: bcrypt 加盐存储(golang.org/x/crypto/bcrypt), 用户改密走 POST /api/v1/auth/change-password, 管理员改密走 POST /api/v1/admin/auth/change-password
- **云端编译工作流**(2026-08 起): 本地仅写代码 → `git push cloud deploy` → 云端 pull + go/npm build + 部署; 不再本地构建产物(见 deploy/AGENTS.md)

## ANTI-PATTERNS (THIS PROJECT)
- **绝不提交**: *.key/*.pem/*.p8, deploy/server.local.json, deploy/*.local*, config.yaml(含 JWT 密钥), install.lock, data/, *.log
- 后台 goroutine 不得用请求 ctx(HTTP 结束即取消 → 消息丢) — 必须 context.Background()
- 不要重建同类型队列实例(消费者订阅旧实例, 重建后消息无人消费) — 已踩坑
- worker 不得在构造时快照 store/registry(Reinit 后写进旧 store → 消息卡 PENDING) — 必须惰性 getter — 已踩坑
- 模板存在"待审核版本"时禁止修改内容
- Gin 根级 catch-all(`/*filepath`)与 /api 路由冲突 panic — 前端托管用 NoRoute 兜底
- NoRoute 排除路径必须精确匹配 `/api` 或 `/api/` 前缀, 不能简单 `HasPrefix("/api")`(误伤 /api-keys 等 SPA 路由) — 已踩坑
- 轮换 admin 随机路径后必须动态注册新前缀路由 + 同步前端相对 base(否则新路径 404)
- smtpSend 必须设连接 deadline(非 SMTP 端口会永久阻塞 worker) — 已踩坑
- Vite 代理只写 `/api/v1`, 勿扩成 `/api`(误伤 /api-keys 等 SPA 路由)
- 云端编译必须前置: 4G swap + `NODE_OPTIONS=--max-old-space-size=2048`(960Mi 内存无 swap 时 node/tsc 直接 OOM Abort) — 已踩坑
- 云端重建数据库后必须 `CREATE EXTENSION IF NOT EXISTS pgcrypto`(DOCKER 库重建后扩展丢失, 重置管理员密码会报 gen_salt 不存在)

## UNIQUE STYLES
- 响应体: `{code:0,message:"ok",data}`; 业务码 0/40000/40100/40300/40400/40900/42900/50000; 消息发送受理返回 HTTP 202
- 消息状态机: PENDING→SENDING→SUCCESS/FAILED/RETRYING/DEAD/CANCELLED; 熔断错误不重试直接 DEAD
- 安装向导: install.lock 门控, 未安装时业务 API 返回 503, 引导 /install; 5 步 POST /api/install/{status,env-check,init,admin,complete}; admin 步骤创建 admins 表首条超管(super_admin)
- 兼容层: Server酱 v1(/api/sc/{key}.send) + v2(/api/sctapi/{key}.send), 返回格式照抄原版

## COMMANDS
```bash
# 后端(本地模式: SQLite + 进程内队列, 内嵌消费者)
go run ./cmd/api                  # :8080
go run ./cmd/web                  # 前端托管(读 config.yaml: 用户中心 :19876 / 管理后台 :19877)
go run ./cmd/redis-mock           # 模拟 Redis :16379(可选)

# 前端
cd web/user && npm run dev        # :5173
cd web/admin && npm run dev       # :5174

# 测试
go test ./...                # 本地验证(仅开发期; 正式验证走云端浏览器)

# 云端编译部署流(2026-08 起, 取代本地构建产物)
git push cloud deploy
ssh mpcloud "sudo bash /opt/messagepusher-src/deploy/cloud-build-deploy.sh"

# 打包可分发安装包(可选, 仍本地打包)
powershell -File deploy/pack-install.ps1    # 输出 dist-install/messagepusher-install-*.tar.gz

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
- git remote: `cloud`(云 bare repo) + `origin`(Gitee 备份); ~/.ssh/config 有 `mpcloud` 别名(86.53.111.210:55244, 走 SOCKS5 代理推送)
- 云部署: Debian12 1C/1G, binary 模式 systemd(api :8080)或单机 compose; PG 用户 mp / Redis 密码见 deploy/server.local.json; 云端编译环境: Go 1.26 + nodejs 18(npm registry 固定 npmmirror)+ 4G swap
- 云端数据重建流程: 停服 → drop/create messagepusher 库 → 删 install.lock → 启动 → 走安装向导(或 SQL 直插超管 + pgcrypto 哈希)
- 端口规划: 主程序 8080(api, 统一路由 / + /{admin}/ + /api) / 用户中心 19876 / 管理后台 19877(cmd/web)
- 前端无 lint/test 脚本, 唯一质量门 = `npm run build`(tsc -b + vite build); npm registry 固定 npmmirror; 无 lockfile 提交
