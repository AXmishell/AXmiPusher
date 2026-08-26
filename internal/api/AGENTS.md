# AGENTS.md: internal/api

HTTP 层。Gin 引擎装配、路由、中间件、handler。依赖注入中心 `*app.App`, 所有 handler 直接接收。

## 文件布局

- `router.go`: 路由装配入口 `NewRouter(a *app.App) *gin.Engine`
- `web.go`: 前端静态托管(SPA, 生产环境)
- `handler/`(18 文件): 业务处理器, 每文件按业务聚合
- `middleware/`: 认证、角色、安装状态中间件

## 路由装配(router.go)

- 顶层 `gin.New()`, 全局挂 `Recovery` + `Logger`
- 安装路由由 `a.InstallRoutes(r)` 注册, 实现在 `internal/app` 包, **不在此处**
- `/api/v1/health` 独立注册, 不在业务组内
- 业务组 `biz := r.Group("/api/v1", RequireInstalled())`, 按业务分子组:

| 子组 | 鉴权 | 说明 |
|---|---|---|
| `/auth` | register/login/login-totp 公开; `/me` 及资料/邮箱/totp 需用户 JWT | 注册校验 confirm_password |
| `/api-keys` `/compat-keys` `/callbacks` `/stats` `/templates` `/pay` `/channels` `/batch-tasks` `/inbox` | RequireAuth | 用户登录态 |
| `/messages` | RequireAuthOrAPIKey | 网页与服务端通用 |
| `/admin/auth` | login/login-totp 公开; `/me` `/profile` `/email` `/change-password` `/totp/*` 需管理员 JWT | 独立管理员体系(admins 表) |
| `/admin/*`(stats/users/reviews/plans/payments/audit-logs/settings...) | RequireAdminAuth | 管理员登录态, **无平台角色门禁**(已由管理员体系取代) |
| `/admin/admins` | RequireAdminAuth + RequireAdminRole(super_admin) | 管理员管理(仅超管) |

- 用户管理(handler/admin.go): GET /users(列表, current/pageSize) + **POST /users(新增, email/username/password 必填, 唯一性 409)** + PUT /users/:id(编辑邮箱/用户名/重置密码, 邮箱唯一排除自身) + PUT /users/:id/status(启禁用); 平台管理员不可编辑/禁用
- 兼容层 `serverchan.RegisterRoutes(r, a)` 在 api 包注册(公开, 走兼容 key 鉴权)
- 支付回调公开且**不经过 RequireInstalled**: `/api/v1/pay/notify`、`/api/v1/pay/return`
- 最后调 `registerWebRoutes`, 挂前端静态托管

## 响应约定(internal/pkg/response)

- 统一体 `{code, message, data}`, 成功 `code=0`
- 业务码: 0 / 40000 参数 / 40100 未认证 / 40300 无权限 / 40400 不存在 / 40900 冲突 / 42900 限流 / 50000 服务端
- handler 一律用 `response.OK/BadRequest/Unauthorized/Forbidden/NotFound/Conflict/RateLimited/ServerError`, 不裸写 JSON
- 例外: 消息发送 `POST /messages` 返回 HTTP 202(受理语义); 支付回调返回纯文本 `success`/`fail`

## 中间件(middleware/auth.go)

- `RequireAuth(getAuth)`: 解析用户 JWT(kind=user) → `CtxUser`
- `RequireAPIKey(getAuth)`: 解析 API Key → `CtxAPIKey` + `CtxTenant`(存 uint64, 即归属用户 ID)
- `RequireAuthOrAPIKey(getAuth)`: 先试 API Key, 再试用户 JWT
- `RequireAdminAuth(getAdminAuth)`: 解析管理员 JWT(kind=admin) → `CtxAdmin`; 用户 token 进不了 admin 端点(kind 校验)
- `RequireAdminRole(roles...)`: 从 CtxAdmin 比对角色(super_admin/admin)
- `RequireRole(roles...)`: 仅限用户 JWT 登录态(已无 platform_admin 用途)
- `RequireInstalled()`: 未安装拒绝业务 API
- 关键: `getAuth`/`getAdminAuth` 是惰性闭包, 容器重建后取最新 AuthService, 勿改成启动时快照
- 上下文键: `CtxUser/CtxTenant(uint64)/CtxAPIKey/CtxAdmin`; 辅助函数 `CurrentUser/CurrentAdmin/CurrentTenantID` 在本包

## TOTP 登录(handler/auth.go + admin_auth.go)

- 第一步 login 返回 `need_totp`+`totp_token`(5 分钟临时凭证); 第二步 `login/totp` 校验验证码后签发正式 token
- totp 三接口(登录态): `setup`(返回 secret/otpauth_url/qr_data_url) → `confirm`(验证码启用) → `disable`(验证码关闭)
- 注册 `confirm_password` 非空时须与 password 一致, 否则 400

## 静态托管(web.go)

- 管理员后台: 显式前缀路由 `/{admin_path}/*filepath`(唯一前缀, 无路由树冲突); handler 用 `app.AdminSPAHandler` **动态校验当前配置路径 — 旧前缀直接 404 废除(不重定向)**
- 用户中心: **NoRoute 兜底**, 静态文件优先, 否则回退 index.html
- NoRoute 内排除 `/api`、`/install`(真实 404); **旧后台前缀兜底: 顶层段为 8-32 位纯字母数字且非当前后台路径且非用户中心 SPA 路由(register/messages/channels/settings 白名单)→ 404**(覆盖重启后旧前缀丢失显式路由落入用户中心 SPA 的漏洞)
- 历史教训: Gin 根级 catch-all(如 `/*filepath`)会吞掉 `/api` 路由, 故用户中心必须走 NoRoute

## 注意

- `handler/rand.go` 命名误导: 实为共享助手 `CurrentUser(c)` 与 `randRead`; 原 randomHex 已删, 随机后台路径统一走 config.GenerateRandomAdminPath
- `PayNotify` 必须 `c.Request.ParseForm()` 后再读 `PostForm`, 否则字段为空
- 列表接口对齐 AntD Pro 分页: query 参数 `current`/`pageSize`(默认 20), 响应含 `total`
- 新增路由: 先在 `biz` 下挂组, 再选鉴权中间件, 勿把公开路由放进业务组
