# AGENTS.md: internal/api

HTTP 层。Gin 引擎装配、路由、中间件、handler。依赖注入中心 `*app.App`, 所有 handler 直接接收。

## 文件布局

- `router.go`: 路由装配入口 `NewRouter(a *app.App) *gin.Engine`
- `web.go`: 生产模式前端静态托管(SPA)
- `handler/`(16 文件): 业务处理器, 每文件按业务聚合
- `middleware/`: 认证、角色、安装状态中间件

## 路由装配(router.go)

- 顶层 `gin.New()`, 全局挂 `Recovery` + `Logger`
- 安装路由由 `a.InstallRoutes(r)` 注册, 实现在 `internal/app` 包, **不在此处**
- `/api/v1/health` 独立注册, 不在业务组内
- 业务组 `biz := r.Group("/api/v1", RequireInstalled())`, 按业务分子组:

| 子组 | 鉴权 | 说明 |
|---|---|---|
| `/auth` | register/login 公开; `/me` 需 JWT | |
| `/api-keys` `/compat-keys` `/callbacks` `/stats` `/templates` `/pay` `/channels` `/batch-tasks` `/inbox` | RequireAuth | 登录态 |
| `/messages` | RequireAuthOrAPIKey | 网页与服务端通用 |
| `/admin` | RequireAuth + RequireRole(platform_admin) | 平台管理, 唯一角色门禁组 |

- 兼容层 `serverchan.RegisterRoutes(r, a)` 在 api 包注册(公开, 走兼容 key 鉴权)
- 支付回调公开且**不经过 RequireInstalled**: `/api/v1/pay/notify`、`/api/v1/pay/return`
- 最后调 `registerWebRoutes`, 挂前端静态托管

## 响应约定(internal/pkg/response)

- 统一体 `{code, message, data}`, 成功 `code=0`
- 业务码: 0 / 40000 参数 / 40100 未认证 / 40300 无权限 / 40400 不存在 / 40900 冲突 / 42900 限流 / 50000 服务端
- handler 一律用 `response.OK/BadRequest/Unauthorized/Forbidden/NotFound/Conflict/RateLimited/ServerError`, 不裸写 JSON
- 例外: 消息发送 `POST /messages` 返回 HTTP 202(受理语义); 支付回调返回纯文本 `success`/`fail`

## 中间件(middleware/auth.go)

- `RequireAuth(getAuth)`: 解析 Bearer JWT → `CtxUser`
- `RequireAPIKey(getAuth)`: 解析 API Key → `CtxAPIKey` + `CtxTenant`
- `RequireAuthOrAPIKey(getAuth)`: 先试 API Key, 再试 JWT
- `RequireRole(roles...)`: 仅限 JWT 登录态
- `RequireInstalled()`: 未安装拒绝业务 API
- 关键: `getAuth` 是惰性闭包 `func() AuthService { return a.Auth }`, 容器重建后取最新 AuthService, 勿改成启动时快照
- 上下文键: `CtxUser/CtxTenant/CtxAPIKey`; 辅助函数 `CurrentUser/CurrentTenant` 也在本包

## 静态托管(web.go)

- 管理员后台: 显式前缀路由 `/{admin_path}/*filepath`(唯一前缀, 无路由树冲突)
- 用户中心: **NoRoute 兜底**, 静态文件优先, 否则回退 index.html
- NoRoute 内排除 `/api`、`/install`、`/{admin_path}/`, 这些路径返回真实 404
- 历史教训: Gin 根级 catch-all(如 `/*filepath`)会吞掉 `/api` 路由, 故用户中心必须走 NoRoute

## 注意

- `handler/rand.go` 命名误导: 实为共享助手 `CurrentUser(c)` 与 `randomHex(n)`, 非随机数业务
- `PayNotify` 必须 `c.Request.ParseForm()` 后再读 `PostForm`, 否则字段为空
- 列表接口对齐 AntD Pro 分页: query 参数 `current`/`pageSize`(默认 20), 响应含 `total`
- 新增路由: 先在 `biz` 下挂组, 再选鉴权中间件, 勿把公开路由放进业务组
