#!/bin/bash
# 云端部署脚本: 从 git clone 的源码组装部署产物并重建容器。
# 用法: sudo bash /opt/axmipusher-src/deploy/cloud-deploy.sh
set -e

STACK=/opt/axmipusher-docker
SRC=/opt/axmipusher-src

echo "[1/4] 拉取最新代码..."
cd "$SRC"
sudo git pull --ff-only origin deploy

echo "[2/4] 同步部署文件与产物到栈目录..."
cd "$STACK"
sudo cp "$SRC/deploy/context/Dockerfile" .
sudo cp "$SRC/deploy/context/docker-compose.single.yml" .
sudo rm -rf web api
sudo mkdir -p web/user web/admin
sudo cp -r "$SRC/deploy/context/web/user/." web/user/
sudo cp -r "$SRC/deploy/context/web/admin/." web/admin/
sudo cp "$SRC/deploy/context/api" api
sudo chmod +x api

echo "[3/4] 重建并重启容器..."
sudo docker compose -f docker-compose.single.yml up -d --build

echo "[4/4] 部署完成: $(curl -s -m 5 http://127.0.0.1:8080/api/v1/health)"
