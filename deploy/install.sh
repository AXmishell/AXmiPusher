#!/bin/bash
# MessagePusher 一键安装脚本。
# 用法: sudo bash install.sh [--mode docker|binary] [--port 8080] [--dir /opt/messagepusher]
# 说明: 解压安装包后在本目录运行; 完成后浏览器访问 http://<IP>:<端口>/install 走 Web 安装向导。
set -e

# ---- 参数解析 ----
MODE="auto"       # auto | docker | binary
PORT="8080"
INSTALL_DIR="/opt/messagepusher"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode) MODE="$2"; shift 2 ;;
    --port) PORT="$2"; shift 2 ;;
    --dir)  INSTALL_DIR="$2"; shift 2 ;;
    *) echo "未知参数: $1"; exit 1 ;;
  esac
done

SRC_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION="$(cat "$SRC_DIR/VERSION" 2>/dev/null || echo "dev")"

echo "=== MessagePusher 安装脚本 v$VERSION ==="
echo "源目录: $SRC_DIR"
echo "安装目录: $INSTALL_DIR"
echo "端口: $PORT"

# ---- 前置检查 ----
[[ $EUID -eq 0 ]] || { echo "[错误] 请用 root 运行 (sudo bash install.sh)"; exit 1; }
[[ -f "$SRC_DIR/api" ]] || { echo "[错误] 缺少 api 二进制, 安装包不完整"; exit 1; }
[[ -f "$SRC_DIR/web/user/index.html" ]] || { echo "[错误] 缺少用户中心前端, 安装包不完整"; exit 1; }
[[ -f "$SRC_DIR/web/admin/index.html" ]] || { echo "[错误] 缺少管理后台前端, 安装包不完整"; exit 1; }
# Windows tar 打包会丢执行位, 统一补上。
chmod +x "$SRC_DIR/api"
chmod +x "$SRC_DIR/install.sh" 2>/dev/null || true

# ---- 模式选择 ----
if [[ "$MODE" == "auto" ]]; then
  if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    MODE="docker"
  else
    MODE="binary"
  fi
fi
echo "运行模式: $MODE"

# ---- 放置文件 ----
mkdir -p "$INSTALL_DIR"
cp -r "$SRC_DIR/api" "$SRC_DIR/web" "$SRC_DIR/Dockerfile" "$SRC_DIR/docker-compose.single.yml" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/api"

install_binary() {
  echo "[binary] 生成 systemd 服务..."
  cat > /etc/systemd/system/messagepusher.service <<EOF
[Unit]
Description=MessagePusher API
After=network.target

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/api
Restart=always
RestartSec=3
Environment=MP_PORT=$PORT
Environment=MP_USER_DIST=$INSTALL_DIR/web/user
Environment=MP_ADMIN_DIST=$INSTALL_DIR/web/admin

[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
  systemctl enable messagepusher >/dev/null 2>&1 || true
  systemctl restart messagepusher
  sleep 2
  if ! systemctl is-active --quiet messagepusher; then
    echo "[错误] 服务启动失败, 查看日志: journalctl -u messagepusher -n 50"
    exit 1
  fi
  echo "[binary] API 服务已启动 (systemd: messagepusher, :$PORT)"

  # 可选: 前端独立端口托管程序(用户中心/管理后台/API 反代)。
  if [[ -x "$INSTALL_DIR/web" ]]; then
    chmod +x "$INSTALL_DIR/web"
    cat > /etc/systemd/system/messagepusher-web.service <<EOF
[Unit]
Description=MessagePusher Web Frontend
After=network.target messagepusher.service

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/web
Restart=always
RestartSec=3
Environment=MP_USER_PORT=19876
Environment=MP_ADMIN_PORT=19877
Environment=MP_API_TARGET=http://127.0.0.1:$PORT
Environment=MP_USER_DIST=$INSTALL_DIR/web/user
Environment=MP_ADMIN_DIST=$INSTALL_DIR/web/admin

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl enable messagepusher-web >/dev/null 2>&1 || true
    systemctl restart messagepusher-web
    echo "[binary] 前端托管已启动 (systemd: messagepusher-web, 用户中心 :19876 / 管理后台 :19877)"
  fi
}

install_docker() {
  echo "[docker] 准备 compose 环境..."
  cd "$INSTALL_DIR"
  # .env 不存在则生成(redis 密码 + admin 路径, 仅存于本机)
  if [[ ! -f .env ]]; then
    ADMIN_PATH="$(head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n' | head -c 16)"
    REDIS_PW="mp-redis-$(head -c 4 /dev/urandom | od -An -tu4 | tr -d ' ')"
    cat > .env <<EOF
REDIS_PASSWORD=$REDIS_PW
MP_ADMIN_PATH=$ADMIN_PATH
EOF
    echo "[docker] .env 已生成 (admin 路径: /$ADMIN_PATH/)"
  fi
  # 改端口
  sed -i "s/\"8080:8080\"/\"$PORT:8080\"/" docker-compose.single.yml
  docker compose -f docker-compose.single.yml up -d --build
  sleep 5
  docker compose -f docker-compose.single.yml ps --format "table {{.Name}}\t{{.Status}}"
  echo "[docker] 容器已启动"
}

case "$MODE" in
  docker) install_docker ;;
  binary) install_binary ;;
esac

# ---- 完成 ----
IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
echo ""
echo "=============================================="
echo " MessagePusher 安装就绪!"
echo ""
echo " 浏览器打开: http://${IP:-<服务器IP>}:$PORT/install"
echo " 按向导完成配置 → 创建管理员"
echo ""
echo " 安装后入口:"
echo "   用户中心   http://${IP:-<服务器IP>}:$PORT/"
echo "   管理后台   http://${IP:-<服务器IP>}:$PORT/<admin随机路径>/"
echo " 卸载:       systemctl disable --now messagepusher  (binary 模式)"
echo "             docker compose -f $INSTALL_DIR/docker-compose.single.yml down  (docker 模式)"
echo "=============================================="
