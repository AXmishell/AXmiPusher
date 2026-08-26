#!/bin/bash
# MessagePusher Release 安装包构建脚本(Linux amd64 单安装包)。
# 用法: bash build-release.sh -v <版本> [-o <输出目录>]
#   -v 必填, 版本号(去前导 v, 如 -v 1.2.3 或 -v v1.2.3 等价)
#   -o 可选, 输出目录(默认 dist-install, 相对仓库根)
# 产物: $OUTDIR/messagepusher-install-$VERSION/ + .tar.gz
# 说明: 供 GitHub Actions 推 v* tag 时自动构建发布; 本机亦可用 bash 直接执行。
set -euo pipefail

# 内存保护: 前端 tsc/vite 构建默认堆过小会 V8 OOM Abort(已踩坑), GitHub runner 内存大但无害。
export NODE_OPTIONS=--max-old-space-size=2048

# ---- 脚本定位: 本文件在 deploy/ 下, 仓库根为其父目录; 之后所有操作均基于仓库根 ----
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(dirname "$SCRIPT_DIR")"
cd "$ROOT"

# ---- 参数解析 ----
VERSION=""
OUTDIR="dist-install"
while [[ $# -gt 0 ]]; do
  case "$1" in
    -v) VERSION="${2#v}"; shift 2 ;;
    -o) OUTDIR="$2"; shift 2 ;;
    *) echo "[错误] 未知参数: $1"; echo "用法: bash build-release.sh -v <版本> [-o <输出目录>]"; exit 1 ;;
  esac
done
[[ -n "$VERSION" ]] || { echo "[错误] 缺少必填参数 -v <版本>"; echo "用法: bash build-release.sh -v <版本> [-o <输出目录>]"; exit 1; }

echo "=== MessagePusher Release 构建 v$VERSION ==="
echo "输出目录: $OUTDIR"

OUT="$OUTDIR/messagepusher-install-$VERSION"
rm -rf "$OUT"
mkdir -p "$OUT"

# 1) 后端 api 二进制(Linux amd64)
echo "=== [1/8] 构建后端 api 二进制 ==="
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$OUT/api" ./cmd/api

# 2) 前端托管程序 web/web-linux(Linux amd64, 独立双端口)
echo "=== [2/8] 构建前端托管程序 web/web-linux ==="
mkdir -p "$OUT/web"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$OUT/web/web-linux" ./cmd/web

# 3) 双前端(user/admin): npm install(无 lockfile 提交, 显式官方 registry) + npm run build
echo "=== [3/8] 构建前端(用户中心 / 管理后台) ==="
for app in user admin; do
  echo "--- web/$app ---"
  (cd "web/$app" && npm install --no-audit --no-fund --registry=https://registry.npmjs.org && npm run build)
  mkdir -p "$OUT/web/$app"
  cp -r "web/$app/dist/." "$OUT/web/$app/"
done

# 4) 拷贝安装脚本与 Docker 部署文件
echo "=== [4/8] 拷贝安装脚本与 Docker 部署文件 ==="
cp deploy/install.sh "$OUT/"
cp deploy/context/Dockerfile "$OUT/"
cp deploy/context/docker-compose.single.yml "$OUT/"

# 5) 版本号与简装说明
echo "=== [5/8] 写入 VERSION 与 README.txt ==="
printf '%s' "$VERSION" > "$OUT/VERSION"
cat > "$OUT/README.txt" <<'EOF'
MessagePusher 安装包
====================
版本: 见 VERSION 文件

安装:
  sudo bash install.sh [--mode docker|binary] [--port 8080]

  默认 auto 模式: 检测到 docker 走容器部署, 否则走 systemd binary 部署。
  完成后浏览器访问 http://<服务器IP>:<端口>/install 走 Web 安装向导。

详见 install.sh 头部注释。
EOF

# 6) 补执行位(GNU tar 保留执行位, 显式补齐更稳; Windows 打包时曾丢位)
echo "=== [6/8] 补齐可执行权限 ==="
chmod +x "$OUT/api" "$OUT/web/web-linux" "$OUT/install.sh"

# 7) 打包
echo "=== [7/8] 打包 tar.gz ==="
tar -czf "$OUTDIR/messagepusher-install-$VERSION.tar.gz" -C "$OUTDIR" "messagepusher-install-$VERSION"

# 8) 输出产物
echo "=== [8/8] 完成 ==="
echo "产物:"
du -h "$OUTDIR/messagepusher-install-$VERSION.tar.gz"
du -sh "$OUT"
