# 打包 MessagePusher 可分发安装包。
# 用法: powershell -File deploy/pack-install.ps1 [-AdminPath b322aa9602150d0c] [-OutDir dist-install]
param(
    [string]$AdminPath = "b322aa9602150d0c",
    [string]$OutDir = "dist-install"
)

$ErrorActionPreference = "Stop"
$root = Split-Path $PSScriptRoot -Parent
$ctxDir = Join-Path $PSScriptRoot "context"

# 版本号: git commit 短号
$version = "1.0.0"
try {
    $v = git -C $root rev-parse --short HEAD 2>$null
    if ($v) { $version = "1.0.0-$v" }
} catch { }

Write-Host "[1/4] 构建产物(build-artifacts + web 前端托管程序)..."
& powershell -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot "build-artifacts.ps1") -AdminPath $AdminPath | Out-Null
Push-Location $root
$env:CGO_ENABLED = "0"; $env:GOOS = "linux"; $env:GOARCH = "amd64"
go build -o bin/web-linux ./cmd/web
Remove-Item Env:CGO_ENABLED, Env:GOOS, Env:GOARCH
Pop-Location

Write-Host "[2/4] 组装安装包目录..."
$pkgDir = Join-Path $root (Join-Path $OutDir "messagepusher-install-$version")
if (Test-Path $pkgDir) { Remove-Item $pkgDir -Recurse -Force }
New-Item -ItemType Directory -Force -Path $pkgDir | Out-Null

Copy-Item (Join-Path $ctxDir "api") $pkgDir
Copy-Item (Join-Path $ctxDir "web") $pkgDir -Recurse
Copy-Item (Join-Path $ctxDir "Dockerfile") $pkgDir
Copy-Item (Join-Path $ctxDir "docker-compose.single.yml") $pkgDir
Copy-Item (Join-Path $PSScriptRoot "install.sh") $pkgDir
Copy-Item (Join-Path $root "bin\web-linux") (Join-Path $pkgDir "web") -Force
Set-Content (Join-Path $pkgDir "VERSION") $version -Encoding ASCII

# 安装说明
$readme = @"
MessagePusher 安装包 v$version
============================

1. 上传本目录到服务器(任意路径)
2. sudo bash install.sh --mode binary --port 8080
   或: sudo bash install.sh --mode docker --port 8080 (需 Docker + Compose)
3. 浏览器访问 http://<IP>:<端口>/install 完成 Web 安装向导

说明:
- binary 模式: 单进程, SQLite + 内置数据库轮询队列 + 可选 Redis, systemd 托管
- docker 模式: 容器编排(api + redis), 自带 .env 生成
- 管理后台为随机路径, 安装完成后在向导页展示
- 凭据不落盘明文: 仅存于服务器本地
"@
Set-Content (Join-Path $pkgDir "README.txt") $readme -Encoding UTF8

Write-Host "[3/4] 打包 tar.gz..."
Push-Location (Join-Path $root $OutDir)
tar -czf "messagepusher-install-$version.tar.gz" "messagepusher-install-$version"
Pop-Location

Write-Host "[4/4] 完成:"
$tarball = Join-Path $root (Join-Path $OutDir "messagepusher-install-$version.tar.gz")
Write-Host "  安装包: $tarball ($([math]::Round((Get-Item $tarball).Length/1MB,1)) MB)"
Write-Host "  测试:   sudo tar -xzf $tarball -C /opt && sudo bash /opt/messagepusher-install-$version/install.sh"
