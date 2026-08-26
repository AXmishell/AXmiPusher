# AGENTS.md - web/ 前端

## 结构

- web/ 下是两个独立 Vite + React18 + TS + antd5 + @ant-design/pro-components 应用:
  - web/user/: 用户中心
  - web/admin/: 管理后台
- 两应用无共享代码: api/client.ts 与 api/notice.tsx 是完整重复拷贝, 改一处必须同步另一处
- 两个 package.json 的 name 分别为 "axmipusher-user"(user) / "axmipusher-admin"(admin), 复制粘贴遗留已修正(勿据此判断应用)
- dev 端口: user=5173, admin=5174; 生产由 Go 后端托管 dist(web.go)

## 各自 src 结构(两应用对称)

- src/api/client.ts: axios 实例 + 拦截器 + request() 解包
- src/api/notice.tsx: 捕获 antd App.useApp() 的 message 实例, 供非组件模块调用 notify()
- src/layouts/MainLayout.tsx: 主布局(侧边菜单 + 内容区)
- src/pages/*: 业务页面, 每页一个文件
- src/App.tsx: 路由表(BrowserRouter + Routes)
- src/main.tsx: 入口, ConfigProvider zhCN + AntApp + MessageHolder

## 页面清单

- user 13 页: Dashboard / SendMessage / Messages / ApiKeys / CompatKeys / Callbacks / Plans / Channels / Inbox / BatchTasks / Profile(账户设置) / Login / Register
- admin 10 页: Users / Admins(管理员管理, 仅超管可见) / Reviews / Plans / Payments / AuditLogs / Settings(系统设置) / Account(账户设置) / Dashboard / Login
- 2026-08 名称/昵称已合并为用户名: User 只保留 nickname(注册时默认 email), 无 name/tenant_name; 账户设置页 Profile(用户中心)/Account(管理后台)展示账号 ID/用户名/邮箱/注册时间/最近登录 IP, 支持改用户名/QQ/邮箱/密码/TOTP

## 关键差异(admin vs user)

- admin: `<BrowserRouter basename={动态取当前路径第一段}>`, 适配部署在任意随机前缀下; 支持运行期轮换 admin 路径(无需重新构建)
- user: 无 base, 直接挂在根路径
- admin 构建用**相对 base('./')**(vite.config.ts: `MP_ADMIN_BASE || './'`), 资源与路由均相对当前前缀; 安装向导从 dist index.html 解析 base 兜底; 旧约定"MP_ADMIN_BASE=/<随机串>/ 且必须与 MP_ADMIN_PATH 一致"已废弃
- token key: user 用 localStorage `mp_token`, admin 用 `mp_admin_token`
- 401 跳登录: user 跳 `/login`(判断 startsWith), admin 跳动态当前前缀 + `/login`(判断 endsWith); admin basename 用 path.ts 的 `resolveAdminBasename`(排除 login/users 等 SPA 路由段, 防 dev 模式误判前缀)
- **TOTP 两步验证**: 两端登录页均两阶段(密码 → `need_totp` → 验证码输入 → `/login/totp`); 账户设置页 TOTP 卡片(后端返回二维码 data URL 直显, 绑定/关闭均需验证码); 注册页有确认密码输入(提交含 confirm_password)

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

## 构建约定

- 无 lint/test 脚本, 唯一质量门 = `npm run build`(`tsc -b && vite build`, 类型检查 + 产物)
- tsconfig strict, 但 `noUnusedLocals/Parameters` 关闭(容忍死代码)
- `.npmrc` 固定 registry=npmmirror(国内网络); 无 lockfile 提交
- `allowScripts.esbuild` 白名单(pnpm-10 风格, 放行 esbuild postinstall)
- **构建在云端完成**(2026-08 起): 本地不跑 build, 由 deploy/cloud-build-deploy.sh 在云端 npm install + build 后部署; 云端需 4G swap + `NODE_OPTIONS=--max-old-space-size=2048`(否则 node/tsc OOM Abort — 已踩坑)

## 维护注意

- 改 client.ts / notice.tsx / MainLayout.tsx 等公共文件时, 记得 user 与 admin 双份同步
