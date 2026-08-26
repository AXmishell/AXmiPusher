# PROJECT KNOWLEDGE BASE — deploy

**Generated:** 2026-08-26 01:58
**Commit:** 820d97a
**Branch:** deploy

## OVERVIEW
部署/安装/分发目录: 本地构建产物 → 打包安装包 / git 推送 → 云端构建或拉产物 → 运行/编排。

## STRUCTURE
```
deploy/
├── build-artifacts.ps1     # 本地构建 api 二进制 + 双前端 dist → context/
├── pack-install.ps1        # 打包可分发安装包 → dist-install/*.tar.gz(含 install.sh + context)
├── install.sh              # 服务器一键安装(docker|binary 双模式, --port/--dir)
├── cloud-deploy.sh         # 云端: git pull → 同步产物到栈目录 → compose 重建
├── docker-deploy.ps1       # 本机驱动云端单机编排(旧流程, 已被 git 流取代)
├── docker-compose.yml      # 生产全栈编排(PG+CH+Kafka+Redis+api+worker, 1G 机器跑不动)
├── server.local.json       # 本机凭据(host/port/key/redis_password, 绝不提交)
└── context/                # 提交入库的部署产物: api 二进制 + web/user|admin dist + Dockerfile + 单机 compose
```

## WHERE TO LOOK
| 任务 | 位置 | 备注 |
|------|------|------|
| 构建部署产物 | build-artifacts.ps1 | -AdminPath 默认 b322aa9602150d0c |
| 打包安装包 | pack-install.ps1 | 输出 messagepusher-install-{ver}.tar.gz |
| 一键安装 | install.sh | binary 模式自动带起 messagepusher-web(19876/19877) |
| 云端更新 | cloud-deploy.sh | 复用 /opt/messagepusher-docker 的 appdata/.env |
| 单机编排 | context/docker-compose.single.yml | api + redis, 需 .env 的 REDIS_PASSWORD |

## CONVENTIONS
- **产物一律本地构建**(Go 交叉编译 + npm build), 服务器/容器内禁止构建 — 1G 内存 OOM
- admin dist 用相对 base('./') 构建(路径无关, 支持任意前缀与运行期轮换); build-artifacts.ps1 不设 MP_ADMIN_BASE — 旧约定"构建 base 必须=MP_ADMIN_PATH(b322aa9602150d0c)"已废弃
- 端口固化: config.yaml 写 server.port=8080 / web.user_port=19876 / web.admin_port=19877(安装向导写入)
- 凭据/运行态绝不提交: server.local.json、*.key/pem/p8、context/.env、context/appdata、dist-install/
- context/ 的 api 二进制与 web dist 需提交入库(部署用), .env 与 appdata 除外
- compose 环境要求: 全栈 docker-compose.yml 需 `.env` 设 `MP_JWT_SECRET`(`${:?}` 缺失即失败); 单机 docker-compose.single.yml 需 `REDIS_PASSWORD`
- Dockerfile 双模式: 根 Dockerfile=源码构建(全栈用, golang:1.26 与 go.mod 1.25 偏差); deploy/context/Dockerfile=二进制拷贝(单机用, 容器内不构建防 OOM)

## ANTI-PATTERNS (已踩坑)
- Windows tar 打包会丢执行位 → install.sh 前置检查用 `-f` 并统一 `chmod +x`
- scp 通配符 + dotfile 上传不可靠 → 用 tar 打包整体传输, 或走 git 流
- git-for-windows 自带 ssh 无法加载 OPENSSH 格式密钥 → core.sshCommand 指系统 OpenSSH + ~/.ssh/config 别名 `mpcloud`
- PowerShell 脚本中文注释需 UTF-8 BOM, 否则 PS 5.1 按 GBK 解析报语法错
- systemd 服务二进制替换需先 stop("Text file busy")
- install.sh 的 binary 模式服务名 `messagepusher` 与旧 systemd 部署重名易混淆

## UNIQUE STYLES
- 部署双模式: docker(binary 产物拷贝式镜像, 防 OOM) / binary(systemd 直接跑 api + 可选 web)
- 云端 git 流: 本地 `git push cloud deploy` → ssh 执行 cloud-deploy.sh(拉取+组装+重建)
- 安装包结构: install.sh + VERSION + README.txt + api + web/{user,admin} + Dockerfile + docker-compose.single.yml

## COMMANDS
```bash
# 本地
powershell -File deploy/build-artifacts.ps1          # 构建产物到 context/
powershell -File deploy/pack-install.ps1             # 打包安装包
git add -A && git commit && git push cloud deploy    # 推送

# 云端
ssh mpcloud "sudo bash /opt/messagepusher-src/deploy/cloud-deploy.sh"
ssh mpcloud "sudo bash <安装包>/install.sh --mode binary --port 8080 --dir /opt/mp"
```

## NOTES
- 1G 内存机器: 云端 go build 可能 OOM, 默认本地构建上传; 云端 apt redis(127.0.0.1:6379, requirepass)与 compose 内置 redis 并存, 端口不冲突
- context.tar.gz 是打包缓存, 不入库
