#!/bin/bash
# 云端编译部署脚本(2026-08 起工作流): git push cloud deploy 后在本机(云端服务器)编译并部署。
# 用法: sudo bash /opt/messagepusher-src/deploy/cloud-build-deploy.sh
# 前置: 云端已装 Go 1.26 + nodejs 18(npmmirror 源)+ 4G swap(960Mi 内存无 swap 会 OOM)。
set -e

SRC=/opt/messagepusher-src
MP=/opt/mp
export GOPROXY=https://goproxy.cn,direct
export PATH=$PATH:/usr/local/go/bin
# 960Mi 物理内存机器上 node/tsc 默认堆过小会 V8 OOM Abort, 必须显式放宽。
export NODE_OPTIONS="--max-old-space-size=2048 --max-semi-space-size=16"

echo "[1/5] 拉取最新代码..."
cd "$SRC"
sudo git pull --ff-only origin deploy

echo "[2/5] 云端编译 Go 后端(1 核, 需数分钟)..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/api-new ./cmd/api
ls -lh /tmp/api-new

echo "[3/5] 云端编译双前端(tsc + vite)..."
cd "$SRC/web/user"
npm install --no-audit --no-fund --registry=https://registry.npmmirror.com > /tmp/npm-user.log 2>&1
npm run build >> /tmp/npm-user.log 2>&1 || { echo 'user 构建失败:'; tail -25 /tmp/npm-user.log; exit 1; }
cd "$SRC/web/admin"
npm install --no-audit --no-fund --registry=https://registry.npmmirror.com > /tmp/npm-admin.log 2>&1
npm run build >> /tmp/npm-admin.log 2>&1 || { echo 'admin 构建失败:'; tail -25 /tmp/npm-admin.log; exit 1; }
echo "  dist: $SRC/web/user/dist + $SRC/web/admin/dist"

echo "[4/5] 停服并替换产物(binary 模式 /opt/mp)..."
sudo systemctl stop messagepusher
sudo cp /tmp/api-new "$MP/api" && sudo chmod +x "$MP/api"
sudo rm -rf "$MP/web/user" "$MP/web/admin"
sudo cp -r "$SRC/web/user/dist" "$MP/web/user"
sudo cp -r "$SRC/web/admin/dist" "$MP/web/admin"

echo "[5/5] 重启并验证..."
sudo systemctl start messagepusher
sleep 3
sudo systemctl is-active messagepusher
curl -s -m 5 http://127.0.0.1:8080/api/v1/health
echo ''
echo "=== 部署完成 ==="

# 备注: 需要清库重装(如大版本重构)时, 先手动执行:
#   sudo systemctl stop messagepusher
#   sudo -u postgres psql -c 'DROP DATABASE IF EXISTS messagepusher;' -c 'CREATE DATABASE messagepusher OWNER mp;'
#   sudo rm -f /opt/mp/install.lock
#   sudo systemctl start messagepusher
#   然后浏览器走 /install 向导(或 SQL 直插超管 + pgcrypto 哈希)。
