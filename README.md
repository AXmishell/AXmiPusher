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
| 部署 | systemd / Docker Compose 双模式;安装包分发(GitHub Releases → install.sh 一键安装) |

## 项目结构

```
AXmiPusher/
├── cmd/             # 3 入口: api(HTTP+内置数据库轮询消费者) / web(前端托管双端口) / redis-mock(开发工具)
├── internal/        # 后端: app(组合根/安装向导) + api + channel + compat + config + db + models + pkg + queue(数据库轮询) + service + store(业务库存储) + worker
├── web/
│   ├── user/        # 用户中心 (Vite+React+AntD Pro)
│   └── admin/       # 管理后台 (随机路径 base, 支持运行期轮换)
├── deploy/          # 安装分发与云端编译部署脚本
├── scripts/         # 本地工具: hook-receiver / mock-smtp
└── openapi.yaml     # API 规范
```

## 部署(安装包分发)

从 GitHub Releases **latest** 自动解析最新版本, 在云端终端一键部署(binary 模式, systemd)。

> 以下命令均在**云端服务器终端**(root)执行, 与本地开发机无依赖。

```bash
# 1. 下载最新安装包(latest 自动解析版本号, 无需手改版本)
cd /opt
VER=$(curl -sI -o /dev/null -w '%{redirect_url}' https://github.com/AXmishell/AXmiPusher/releases/latest | sed 's|.*/||')
curl -sL -o axmipusher-install-${VER#v}.tar.gz \
  "https://github.com/AXmishell/AXmiPusher/releases/download/$VER/axmipusher-install-${VER#v}.tar.gz"

# 2. 停旧服务 + 清理旧配置(全新安装时)
systemctl stop axmipusher axmipusher-web 2>/dev/null
rm -f /opt/axmipusher/config.yaml /opt/axmipusher/install.lock

# 3. 解压 + 一键安装(自动生成 systemd 服务并启动)
tar -xzf axmipusher-install-*.tar.gz
cd axmipusher-install-*/
bash install.sh --mode binary --dir /opt/axmipusher

# 4. 浏览器访问 http://<IP>:8080/install 走安装向导
#    (环境检查 → 配置数据库/Redis → 创建平台超管 → 完成)
#    ⚠️ 向导完成后必须重启服务, 管理后台随机路径路由才会注册(v1.0.1 起生效):
systemctl restart axmipusher

# 5. 验证
curl http://127.0.0.1:8080/api/v1/health        # {"installed":true}
curl -I http://127.0.0.1:8080/<admin随机路径>/  # 返回管理后台
```

部署后入口(端口固化):

| 服务 | systemd 单元 | 端口 | 说明 |
|---|---|---|---|
| api(主程序) | `axmipusher` | 8080 | 统一入口: 用户中心 `/` + 管理后台 `/{随机路径}/` |
| 前端托管(cmd/web) | `axmipusher-web` | 19876 / 19877 | 独立端口托管用户中心 / 管理后台 |

> **全新安装建议重置数据**(云端终端): PostgreSQL 重建库 + `CREATE EXTENSION pgcrypto`, Redis `FLUSHALL`:
> ```bash
> sudo -u postgres psql -c 'DROP DATABASE IF EXISTS axmipusher;' -c 'CREATE DATABASE axmipusher OWNER mp;'
> sudo -u postgres psql -d axmipusher -c 'CREATE EXTENSION IF NOT EXISTS pgcrypto;'
> redis-cli -a '<Redis密码>' --no-auth-warning FLUSHALL
> ```
> 也可本地打包安装包:`powershell -File deploy/pack-install.ps1` → 输出 `dist-install/axmipusher-install-*.tar.gz`。

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
