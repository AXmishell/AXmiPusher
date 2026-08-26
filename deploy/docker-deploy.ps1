# AXmiPusher 单机 Docker 一键部署脚本。
# 流程: 构建产物(Go 交叉编译 + 双前端) → 组装部署目录 → 上传服务器 → compose up。
# 前置: 服务器已装 Docker + Compose 插件; deploy/server.local.json 已配置。
param(
    [string]$AdminPath = "b322aa9602150d0c"
)

$ErrorActionPreference = "Stop"
$root = Split-Path $PSScriptRoot -Parent
$cfg = Get-Content (Join-Path $PSScriptRoot "server.local.json") -Raw | ConvertFrom-Json
$keyPath = Join-Path $root $cfg.key
$target = "$($cfg.user)@$($cfg.host)"
$ctxDir = Join-Path $PSScriptRoot "context"
$remoteDir = "/opt/axmipusher-docker"

Write-Host "[1/6] 构建 Linux 二进制..."
Push-Location $root
$env:CGO_ENABLED = "0"; $env:GOOS = "linux"; $env:GOARCH = "amd64"
go build -o bin/api-linux ./cmd/api
Remove-Item Env:CGO_ENABLED, Env:GOOS, Env:GOARCH
Pop-Location

Write-Host "[2/6] 构建用户中心前端..."
Push-Location (Join-Path $root "web\user")
npm run build
Pop-Location

Write-Host "[3/6] 构建管理员后台(base=/$AdminPath/)..."
Push-Location (Join-Path $root "web\admin")
$env:MP_ADMIN_BASE = "/$AdminPath/"
npm run build
Remove-Item Env:MP_ADMIN_BASE
Pop-Location

Write-Host "[4/6] 组装部署目录..."
# 清理旧产物(保留 appdata/.env/Dockerfile/compose)。
Get-ChildItem $ctxDir -Exclude "appdata", ".env", "Dockerfile", "docker-compose.single.yml" | Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
Copy-Item (Join-Path $root "bin\api-linux") (Join-Path $ctxDir "api")
New-Item -ItemType Directory -Force -Path (Join-Path $ctxDir "web\user"), (Join-Path $ctxDir "web\admin") | Out-Null
Copy-Item (Join-Path $root "web\user\dist\*") (Join-Path $ctxDir "web\user\") -Recurse -Force
Copy-Item (Join-Path $root "web\admin\dist\*") (Join-Path $ctxDir "web\admin\") -Recurse -Force

# 生成 .env(密码随机, 首次生成后保留)
$envFile = Join-Path $ctxDir ".env"
if (-not (Test-Path $envFile)) {
    $pw = "mp-redis-" + (Get-Random -Minimum 100000 -Maximum 999999)
    Set-Content $envFile "REDIS_PASSWORD=$pw`nMP_ADMIN_PATH=$AdminPath" -Encoding ASCII
    Write-Host "    .env 已生成(密码仅存于此文件)"
}

Write-Host "[5/6] 上传到 $($cfg.host):$remoteDir ..."
& ssh -i $keyPath -p $cfg.port -o ConnectTimeout=20 $target "sudo mkdir -p $remoteDir && sudo chmod -R 777 $remoteDir"
scp -i $keyPath -P $cfg.port -o ConnectTimeout=30 -r "$ctxDir\*" "$($target):$remoteDir/" | Out-Null

Write-Host "[6/6] 迁移数据 + 停止旧部署并启动 compose..."
& ssh -i $keyPath -p $cfg.port -o ConnectTimeout=20 $target @"
cd $remoteDir
# 迁移既有数据(首次部署时): 保留已安装状态(用户/消息/admin 路径)
if [ ! -d appdata/data ]; then
    sudo mkdir -p appdata/data
    sudo cp /opt/axmipusher/config.yaml appdata/ 2>/dev/null || true
    sudo cp /opt/axmipusher/install.lock appdata/ 2>/dev/null || true
    sudo cp -r /opt/axmipusher/data/. appdata/data/ 2>/dev/null || true
    echo "数据已迁移到 appdata/"
fi
sudo systemctl stop axmipusher 2>/dev/null; sudo systemctl disable axmipusher 2>/dev/null
sudo systemctl stop redis-server 2>/dev/null; sudo systemctl disable redis-server 2>/dev/null
sudo docker compose -f docker-compose.single.yml up -d --build
sleep 8
sudo docker compose -f docker-compose.single.yml ps
"@

Write-Host "部署完成, 检查: curl http://mp.gpcn.cc:8080/api/v1/health"
