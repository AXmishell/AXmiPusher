# PROJECT KNOWLEDGE BASE — deploy

**Generated:** 2026-08-26 21:40
**Commit:** a6c23b1
**Branch:** deploy

## OVERVIEW
部署/安装/分发目录: 本地仅写代码 → git 推送 → 云端编译部署(cloud-build-deploy.sh); 发布走 GitHub Actions(推 v* tag → build-release.sh 自动打包发布)。

## STRUCTURE
```
deploy/
├── build-release.sh      # Release 安装包构建(Linux amd64, GitHub Actions 推 v* tag 时调用; 本机 bash 亦可跑)
├── cloud-build-deploy.sh # 云端编译部署(2026-08 起主流程): pull → go/npm build → 替换 → 重启
├── build-artifacts.ps1   # 本地构建 api 二进制 + 双前端 dist → context/(仅安装包打包用, 常规部署不再需要)
├── pack-install.ps1      # 本地打包可分发安装包 → dist-install/*.tar.gz(Windows 专用, 等价 build-release.sh)
├── install.sh            # 服务器一键安装(docker|binary 双模式, --port/--dir)
├── cloud-deploy.sh       # 旧云端流程(git pull → 同步产物到 docker 栈, 已被 cloud-build-deploy.sh 取代)
├── docker-deploy.ps1     # 本机驱动云端单机编排(旧流程, 已被 git 流取代)
├── docker-compose.yml    # 生产全栈编排(PG+Redis+api, 1G 机器跑不动)
├── server.local.json     # 本机凭据(host/port/key/redis_password, 绝不提交)
└── context/              # 安装包源: Dockerfile + docker-compose.single.yml 已入库(git add -f); api 二进制/web dist 为本地构建产物, 不入库
```

## WHERE TO LOOK
| 任务 | 位置 | 备注 |
|------|------|------|
| 发布 Release 安装包 | build-release.sh + .github/workflows/release.yml | 推 v* tag 自动构建并发布到 GitHub Releases(版本号 = tag 名去 v, 语义化 SemVer) |
| 云端编译部署 | cloud-build-deploy.sh | 主流程: 本地 push → ssh 执行 |
| 打包安装包(本地) | pack-install.ps1 | 输出 messagepusher-install-{ver}.tar.gz(需先 build-artifacts.ps1) |
| 一键安装 | install.sh | binary 模式自动带起 messagepusher-web(19876/19877, 二进制在 web/web-linux) |
| 清库重装 | cloud-build-deploy.sh 尾部注释 | 停服 → drop/create 库 → 删 install.lock → 向导 |

## CONVENTIONS
- **云端编译工作流**(2026-08 起): 本地仅写代码, `git push cloud deploy` → `sudo bash /opt/messagepusher-src/deploy/cloud-build-deploy.sh`。产物不再本地构建。
- 云端编译前置(960Mi 1 核机器, 已踩坑): **4G swap** + `NODE_OPTIONS=--max-old-space-size=2048`(node 默认堆过小 → V8 OOM Abort); Go 1.26 + nodejs 18 + npm registry 固定 npmmirror。
- admin dist 用相对 base('./') 构建(路径无关, 支持任意前缀与运行期轮换) — 旧约定"构建 base 必须=MP_ADMIN_PATH"已废弃。
- 端口固化: config.yaml 写 server.port=8080 / web.user_port=19876 / web.admin_port=19877(安装向导写入)。
- 凭据/运行态绝不提交: server.local.json、*.key/pem/p8、context/.env、context/appdata、dist-install/。
- 部署目标: binary 模式 systemd(`/opt/mp/api`, web 由主程序托管), 见 /opt/mp/config.yaml。

## ANTI-PATTERNS (已踩坑)
- Windows tar 打包会丢执行位 → install.sh 前置检查用 `-f` 并统一 `chmod +x`
- scp 通配符 + dotfile 上传不可靠 → 用 tar 打包整体传输, 或走 git 流
- git-for-windows 自带 ssh 无法加载 OPENSSH 格式密钥 → core.sshCommand 指系统 OpenSSH + ~/.ssh/config 别名 `mpcloud`
- PowerShell 脚本中文注释需 UTF-8 BOM, 否则 PS 5.1 按 GBK 解析报语法错
- systemd 服务二进制替换需先 stop("Text file busy")
- install.sh 的 binary 模式服务名 `messagepusher` 与旧 systemd 部署重名易混淆

## UNIQUE STYLES
- 部署双模式: docker(binary 产物拷贝式镜像, 防 OOM) / binary(systemd 直接跑 api + 可选 web)
- 云端 git 流: 本地 `git push cloud deploy` → ssh 执行 cloud-build-deploy.sh(云端编译 + 部署)
- Release 流: `git push github --tags` → GitHub Actions 构建发布(build-release.sh, 包名/目录名/VERSION 取 tag 名去 v)
- 安装包结构: install.sh + VERSION + README.txt + api + web/{user,admin,web-linux} + Dockerfile + docker-compose.single.yml

## COMMANDS
```bash
# 云端编译部署(主流程)
git push cloud deploy
ssh mpcloud "sudo bash /opt/messagepusher-src/deploy/cloud-build-deploy.sh"

# GitHub 备份/CI(.github/workflows/ci.yml, push main/deploy 自动编译检查)
git push github deploy:main

# 发布 Release(推 v* tag, GitHub Actions 自动构建并发布安装包)
git tag v1.0.0 && git push github v1.0.0

# 本地打包可分发安装包(可选, 需 bash 或 Windows PS)
bash deploy/build-release.sh -v 1.0.0     # Linux/macOS
powershell -File deploy/build-artifacts.ps1 -AdminPath b322aa9602150d0c; powershell -File deploy/pack-install.ps1  # Windows

# 云端清库重装(大版本重构时)
#   停服 → drop/create messagepusher 库 → 删 install.lock → 启动 → 走 /install 向导
```

## NOTES
- 云端编译需 4G swap(已建 /swapfile, 开机自挂); 960Mi 物理内存下 node/tsc 必须 NODE_OPTIONS=--max-old-space-size=2048, 否则 V8 OOM Abort — 已踩坑
- 云端 apt redis(127.0.0.1:6379, requirepass)与 compose 内置 redis 并存, 端口不冲突
- context.tar.gz 是打包缓存, 不入库; deploy/context/.env 由 .gitignore 忽略; context/api 二进制已移出跟踪(54MB 超 GitHub 限制); **context 的 Dockerfile 与 docker-compose.single.yml 已 git add -f 入库**(安装包源, 需保持跟踪)
- build-release.sh 用 GNU tar 保留执行位, 但显式 chmod +x api/web/web-linux 更稳; install.sh 的 web 服务判定必须是 `web/web-linux`(web 是目录)
- 云端重建数据库后必须 `CREATE EXTENSION IF NOT EXISTS pgcrypto`(否则重置管理员密码报 gen_salt 不存在)
- git remote: `cloud`(云 bare repo, 部署主通道) + `github`(GitHub AXmishell/AXmiPusher, main 分支, 备份+CI+Release); 敏感文件(config.yaml.bak 等)绝不推 GitHub — 已踩坑
