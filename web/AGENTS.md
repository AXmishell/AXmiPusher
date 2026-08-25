# AGENTS.md - web/ 前端

## 结构

- web/ 下是两个独立 Vite + React18 + TS + antd5 + @ant-design/pro-components 应用:
  - web/user/: 用户中心
  - web/admin/: 管理后台
- 两应用无共享代码: api/client.ts 与 api/notice.tsx 是完整重复拷贝, 改一处必须同步另一处
- 两个 package.json 的 name 都是 "messagepusher-user"(复制粘贴遗留), 勿据此判断应用
- dev 端口: user=5173, admin=5174; 生产由 Go 后端托管 dist(web.go)

## 各自 src 结构(两应用对称)

- src/api/client.ts: axios 实例 + 拦截器 + request() 解包
- src/api/notice.tsx: 捕获 antd App.useApp() 的 message 实例, 供非组件模块调用 notify()
- src/layouts/MainLayout.tsx: 主布局(侧边菜单 + 内容区)
- src/pages/*: 业务页面, 每页一个文件
- src/App.tsx: 路由表(BrowserRouter + Routes)
- src/main.tsx: 入口, ConfigProvider zhCN + AntApp + MessageHolder

## 页面清单

- user 12 页: Dashboard / SendMessage / Messages / ApiKeys / CompatKeys / Callbacks / Plans / Channels / Inbox / BatchTasks / Login / Register
- admin 9 页: Tenants / Users / Reviews / Plans / Payments / AuditLogs / Settings / Dashboard / Login

## 关键差异(admin vs user)

- admin: `<BrowserRouter basename={import.meta.env.BASE_URL}>`, 适配部署在随机路径下的场景
- user: 无 base, 直接挂在根路径
- admin 构建必须设环境变量 `MP_ADMIN_BASE=/<随机串>/`(vite.config.ts 的 base), 且与后端 MP_ADMIN_PATH 一致
- token key: user 用 localStorage `mp_token`, admin 用 `mp_admin_token`
- 401 跳登录: user 跳 `/login`(判断 startsWith), admin 跳 `${BASE_URL}login`(判断 endsWith)

## API 调用约定

- axios baseURL `/api/v1`, 同源请求, 不写死域名
- 请求拦截器自动加 `Authorization: Bearer <token>`(从 localStorage 读对应 key)
- 401 响应: 自动清 token 并跳登录页
- 后端统一响应 `{code, message, data}`: request() 解包返回 data, code!=0 抛 Error
- 业务错误统一走 notify() 弹 antd message(见 notice.tsx 的 MessageHolder)
- ProTable 的 request 必须返回 `{data, total, success}`, 分页参数 `current` / `pageSize`
- UI 全中文: antd ConfigProvider locale=zhCN + dayjs locale zh-cn

## 开发代理

- vite 只代理 `/api/v1` → http://localhost:8080
- 勿扩成 `/api`, 会误伤 `/api-keys` 等 SPA 前端路由(已踩坑)

## 维护注意

- 改 client.ts / notice.tsx / MainLayout.tsx 等公共文件时, 记得 user 与 admin 双份同步
