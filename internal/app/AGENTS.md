# AGENTS.md: internal/app — 组合根/DI 容器 + 安装向导

全仓依赖中心。`App` 结构体装配配置、DB、存储、队列、服务与渠道; 同时承载安装向导 HTTP 处理器(install.go)与前端静态托管路由注册。

## 文件地图

- `app.go`: `App` 结构体 + `New` + `Build`(按配置装配全部组件)+ `Reinit`(安装后重建)+ `setupRedis` + `StartConsumer` + `Close`
- `install.go`: 安装向导 5 步路由 + 全部 handler + helpers(351 行, 全仓最大文件)
- `embed.go` / `install.html`: 安装向导前端页面(go:embed)

## 核心模式

### Build 装配顺序(不可乱)
1. `db.Open` → GORM(生产 PG / 本地 SQLite)
2. `store`(sqlite/clickhouse)→ 消息记录存储
3. `queue`(inprocess/kafka)→ 消息队列
4. 服务: Auth/Limiter/Messages/Templates/Settings/Pay/Batch
5. `setupRedis`: Redis 探活失败按配置降级内存
6. 渠道注册: Breaker(内存/Redis)→ Registry → 5 个 Sender; 最后把 HasChannel/IsAvailable 注入 Messages(快速失败 + 熔断降级路由)

### Reinit(安装程序写入配置后调用)
- 保留**同类型**队列实例(`keepQueue`): 消费者已订阅旧实例, 重建 → 消息无人消费 — 已踩坑
- 只重建 Store/服务/渠道, Queue 复用; 然后 `a.Messages.SetQueue(keepQueue)` 回绑

### Router 注入
- `App.Router` 由 `cmd/api` 在 `NewRouter` 后注入(`a.Router = router`)
- 用途: 轮换 admin 路径时 `RegisterAdminSPA(newPath, adminDist)` 动态注册新前缀路由(否则新路径 404)

## 安装向导(install.go)

- 路由: `POST /api/install/{status, env-check, init, admin, complete}` + `GET /install`(HTML)
- 状态机: `config.IsInstalled()`(install.lock 存在)门控; 未安装时业务 API 503
- `init` 流程: 生成 JWT 密钥 → **admin path 从 admin dist index.html 解析**(DetectAdminBase, 与构建 base 一致, 否则管理后台 404)→ 写 config.yaml → `Reinit` → seedPlans(默认 3 套餐)
- `admin`: 创建 admins 表首条超管(super_admin, 邮箱/昵称/密码≥8, bcrypt); admins 表已有数据则 Conflict
- `complete`: `config.MarkInstalled()` 写 install.lock
- 老数据迁移: db 包 `migrateLegacyAdmins` 启动时把 users.role=platform_admin 迁入 admins(超管)后从 users 删除(仅旧版升级)
- 端口固化: server.port=8080 / web.user_port=19876 / web.admin_port=19877 / api_target 写死

## 关键函数

| 函数 | 作用 | 注意 |
|------|------|------|
| `DetectAdminBase(distDir)` | 从 admin dist index.html 正则解析 base 路径 | 相对 base('./') 构建时匹配不到 → 回退默认 |
| `RegisterAdminSPA(prefix, distDir)` | 动态注册 admin 前缀 SPA 静态托管 | Router 为 nil 时跳过(worker 进程) |
| `Reinit(cfg)` | 安装后重建组件 | 保留同类型队列, 勿重建 |

## 已踩坑

- Reinit 不得重建同类型队列(消费者订阅旧实例 → 无人消费)
- admin path 必须与构建 base 一致(DetectAdminBase 兜底, 否则管理后台 404)
- 轮换路径后必须 RegisterAdminSPA 新前缀(仅改 config 不够, 路由树不自动更新)
- 安装向导用 `randomHex(32)` 生成 JWT 密钥, 勿硬编码/复用
- 旧库启动清理(migrateLegacySchema, db 包): users 表残留 tenant_id/name 列会致新代码插入 NOT NULL 失败, 启动自动 DROP(SQLite 先删索引); 已踩坑
