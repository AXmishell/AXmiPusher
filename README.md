# AXmiPusher 消息推送平台

统一受理消息 → 队列 → 多渠道发送(Webhook / Email / APNs / FCM / 站内信)→ 回执统计的一站式消息推送平台。
Go 后端(单仓库, api 单进程内置数据库轮询消费者)+ 双独立 React 前端(用户中心 / 管理后台),支持 SQLite 单机运行与 PostgreSQL 生产部署。

## 功能特性

- **多渠道发送**: Webhook、Email(SMTP)、APNs、FCM、站内信,支持自动渠道路由与熔断降级
- **发送治理**: 幂等(request_id + DB 唯一索引 + Redis 加速)、限流(内存/Redis 双实现)、消息模板(含审核流)
- **兼容层**: Server酱 v1(`/api/sc/{key}.send`)/ v2(`/api/sctapi/{key}.send`)零改动接入
- **回调订阅**: 消息状态变更 Webhook 通知
- **批量任务**: 大名单分批发送,进度实时可见,支持取消
- **双端分离**: 用户中心(消息/Key/模板/渠道/订阅)+ 管理后台(用户/审核/套餐/支付/审计/系统设置),JWT `kind` 双向隔离
- **账户设置**: 用户中心与管理后台均可修改昵称/邮箱/QQ/密码,展示注册时间与最近登录 IP
- **安装向导**: 5 步可视化安装(环境检查 → 初始化 → 创建超管 → 完成),未安装自动门控

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.25 + Gin + GORM,数据库驱动(PostgreSQL / SQLite);Redis 可选(限流/熔断/幂等) |
| 前端 | Vite + React 18 + TypeScript + Ant Design 5(ProComponents),两个独立应用 |
| 部署 | systemd / Docker Compose 双模式;云端编译工作流(git push → 云端 go/npm build) |

## 项目结构

```
messagepusher/
├── cmd/             # 3 入口: api(HTTP+内置数据库轮询消费者) / web(前端托管双端口) / redis-mock(开发工具)
├── internal/        # 后端: app(组合根/安装向导) + api + channel + compat + config + db + models + pkg + queue(数据库轮询) + service + store(业务库存储) + worker
├── web/
│   ├── user/        # 用户中心 (Vite+React+AntD Pro)
│   └── admin/       # 管理后台 (随机路径 base, 支持运行期轮换)
├── deploy/          # 安装分发与云端编译部署脚本
├── scripts/         # 本地工具: hook-receiver / mock-smtp
└── openapi.yaml     # API 规范
```

## 快速开始(SQLite 单机)

```bash
# 后端(SQLite + 内置数据库轮询消费者)
go run ./cmd/api                  # :8080, 首次访问 /install 走安装向导

# 前端(可选, 生产由主程序托管 dist)
cd web/user && npm run dev        # :5173
cd web/admin && npm run dev       # :5174

# 测试
go test ./...
```

## 部署(云端编译工作流)

本地仅编写代码,产物在云端编译:

```bash
git push cloud deploy
ssh mpcloud "sudo bash /opt/messagepusher-src/deploy/cloud-build-deploy.sh"
```

详见 [`deploy/AGENTS.md`](deploy/AGENTS.md)。也可本地打包可分发安装包:

```bash
powershell -File deploy/pack-install.ps1    # 输出 dist-install/messagepusher-install-*.tar.gz
```

## 账号体系

| 体系 | 数据表 | 入口 | 说明 |
|---|---|---|---|
| 用户中心 | `users` | `/` | 开放注册, 承载配额/套餐, JWT kind=user |
| 管理后台 | `admins` | `/{随机路径}/` | 安装向导创建超管, 支持多管理员, JWT kind=admin |

两套体系**双向隔离**:用户 token 进不了管理端点,管理员 token 进不了用户端点。服务端调用另有 API Key(`mp_` 前缀, SHA-256 哈希存库)。

## 配置

- 配置优先级: `MP_*` 环境变量 > `config.yaml` > 默认值
- `config.yaml` 由安装向导生成于工作目录,缺失 = 未安装状态(业务 API 返回 503)
- 端口固化: 主程序 8080 / 用户中心 19876 / 管理后台 19877

## 相关文档

- [`AGENTS.md`](AGENTS.md) — 项目知识库(代码地图/约定/踩坑记录)
- [`deploy/AGENTS.md`](deploy/AGENTS.md) — 部署与安装分发
- [`test.md`](test.md) — 测试流程
- [`openapi.yaml`](openapi.yaml) — API 规范
