#!/bin/bash
# AXmiPusher 一键部署脚本(基于 GitHub Releases 安装包, binary 模式 systemd)。
# 自动完成: 解析 latest 版本 → 下载安装包 → 停旧服务 → 清理旧配置 → 解压 → install.sh --mode binary → 启动。
# 用法(root):
#   curl -sSL https://raw.githubusercontent.com/AXmishell/AXmiPusher/main/deploy/one-click-install.sh | sudo bash
# 可调环境变量:
#   INSTALL_DIR   安装目录(默认 /opt/axmipusher, 对应 install.sh 的 --dir)
#   KEEP_CONFIG   置 1 保留旧 config.yaml/install.lock(升级安装; 默认 0 = 全新安装清空配置)
set -euo pipefail

REPO="AXmishell/AXmiPusher"
INSTALL_DIR="${INSTALL_DIR:-/opt/axmipusher}"
KEEP_CONFIG="${KEEP_CONFIG:-0}"

[[ $EUID -eq 0 ]] || { echo "[错误] 请用 root 运行 (sudo bash one-click-install.sh)"; exit 1; }

echo "=== AXmiPusher 一键部署 ==="
echo "安装目录: $INSTALL_DIR"

# ---- 1. 解析 latest 版本号 ----
VER="$(curl -sI -o /dev/null -w '%{redirect_url}' "https://github.com/$REPO/releases/latest" | sed 's|.*/||')"
[[ -z "$VER" ]] && { echo "[错误] 解析 GitHub latest 版本失败, 请检查网络"; exit 1; }
echo "最新版本: $VER"

# ---- 2. 下载安装包 ----
PKG="axmipusher-install-${VER#v}.tar.gz"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT
cd "$TMP_DIR"
echo "下载: $PKG"
curl -sL --fail -o "$PKG" "https://github.com/$REPO/releases/download/$VER/$PKG" || {
  echo "[错误] 下载安装包失败"; exit 1;
}

# ---- 3. 停旧服务 ----
systemctl stop axmipusher axmipusher-web 2>/dev/null || true
echo "旧服务已停止"

# ---- 4. 清理旧配置(全新安装; KEEP_CONFIG=1 时保留) ----
if [[ "$KEEP_CONFIG" != "1" ]]; then
  rm -f "$INSTALL_DIR/config.yaml" "$INSTALL_DIR/install.lock"
  echo "旧配置已清理(config.yaml / install.lock)"
else
  echo "保留旧配置(升级模式)"
fi

# ---- 5. 解压 + 安装(binary 模式, 自动生成 systemd 服务并启动) ----
tar -xzf "$PKG"
cd "axmipusher-install-${VER#v}"
bash install.sh --mode binary --dir "$INSTALL_DIR"

# ---- 6. 完成提示 ----
IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
echo ""
echo "=============================================="
echo " 一键部署完成!"
echo ""
echo " 1. 浏览器打开: http://${IP:-<服务器IP>}:8080/install 走安装向导"
echo "    (环境检查 → 配置数据库/Redis → 创建平台超管 → 完成)"
echo " 2. 向导完成后直接访问管理后台(当前版本无需重启)"
echo "    用户中心   http://${IP:-<服务器IP>}:8080/"
echo "    管理后台   http://${IP:-<服务器IP>}:8080/<admin随机路径>/"
echo " 3. 验证: curl http://127.0.0.1:8080/api/v1/health"
echo "=============================================="
