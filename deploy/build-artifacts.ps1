# 本地构建部署产物脚本: 生成 deploy/context/ 下的 api + 双前端 dist(供 git 提交)。
# 用法: powershell -File deploy/build-artifacts.ps1 [-AdminPath b322aa9602150d0c]
param(
    [string]$AdminPath = "b322aa9602150d0c"
)

$ErrorActionPreference = "Stop"
$root = Split-Path $PSScriptRoot -Parent
$ctxDir = Join-Path $PSScriptRoot "context"

Write-Host "[1/3] 构建 Linux 二进制..."
Push-Location $root
$env:CGO_ENABLED = "0"; $env:GOOS = "linux"; $env:GOARCH = "amd64"
go build -o bin/api-linux ./cmd/api
Remove-Item Env:CGO_ENABLED, Env:GOOS, Env:GOARCH
Pop-Location

Write-Host "[2/3] 构建双前端..."
Push-Location (Join-Path $root "web\user")
npm run build
Pop-Location
Push-Location (Join-Path $root "web\admin")
# 注意: 不再设置 MP_ADMIN_BASE —— admin 构建用相对 base('./'),
# 资源与路由相对当前前缀, 支持任意部署路径与运行期轮换 admin 路径。
npm run build
Pop-Location

Write-Host "[3/3] 组装到 deploy/context/..."
Copy-Item (Join-Path $root "bin\api-linux") (Join-Path $ctxDir "api") -Force
Remove-Item (Join-Path $ctxDir "web\user"), (Join-Path $ctxDir "web\admin") -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path (Join-Path $ctxDir "web\user"), (Join-Path $ctxDir "web\admin") | Out-Null
Copy-Item (Join-Path $root "web\user\dist\*") (Join-Path $ctxDir "web\user\") -Recurse -Force
Copy-Item (Join-Path $root "web\admin\dist\*") (Join-Path $ctxDir "web\admin\") -Recurse -Force

Write-Host "完成. 下一步: git add -A && git commit && git push cloud deploy"
