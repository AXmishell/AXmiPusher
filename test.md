# MessagePusher 测试流程

> 目标: 从**空数据库**开始, 走完整安装向导, 逐页测试前端全部按钮/交互。
> 环境: 默认云端生产 `http://mp.gpcn.cc:8080`(本地可用 `http://localhost:8080` 替换)。
> 前置: 已部署最新代码(`git push origin deploy` 后云端 `cloud-deploy.sh`)。

---

## 0. 前置条件

- [ ] 云端服务运行中: `curl http://mp.gpcn.cc:8080/api/v1/health` 返回 `installed:true/false` 均可
- [ ] 有服务器 SSH 权限(`mpcloud` 别名) 或 本地目录操作权限
- [ ] 准备测试账号密码(安装向导创建平台管理员)

---

## 1. 阶段一: 初始化数据库(清空)

> ⚠️ 清空后旧数据不可恢复, 确认在测试环境执行。

### 1.1 清空 PostgreSQL

```bash
ssh mpcloud "sudo -u postgres psql -c 'DROP DATABASE IF EXISTS messagepusher;' \
  && sudo -u postgres psql -c 'CREATE DATABASE messagepusher OWNER mp;'"
```

### 1.2 清空 Redis

```bash
ssh mpcloud "redis-cli -a '<密码>' --no-auth-warning FLUSHALL"
```

### 1.3 清空本地模式 SQLite(如用本地测试)

```powershell
Remove-Item data\messagepusher.db* -Force
```

### 1.4 移除安装锁与配置(重置为未安装态)

```bash
# 云端 binary 模式 / 本地
ssh mpcloud "rm -f /opt/mp-main/install.lock /opt/mp-main/config.yaml"
# 或本地
Remove-Item install.lock, config.yaml -Force
```

- [ ] 验证: 重启服务后 `POST /api/install/status` → `{"installed":false}`; 业务 API 返回 503

---

## 2. 阶段二: 运行安装程序

### 2.1 启动服务

```bash
# 云端(binary 或 compose) / 本地
go run ./cmd/api          # :8080
```

### 2.2 浏览器访问安装向导

打开 `http://<域名>:8080/install`

| 步骤 | 操作 | 预期 |
|---|---|---|
| step0 欢迎 | 点击【开始安装】 | 进入环境检查 |
| step1 环境检查 | 等待自动检测 | SQLite 目录 / PG 连接均 ✓ |
| step2 基础配置 | 选 PG 或 SQLite + 填 Redis(可选) + 保留天数 | 点击【初始化】→ 进入 step3 |
| step3 创建管理员 | 填邮箱/昵称/密码(≥8位) | 点击【创建账号】→ 进入 step4 |
| step4 完成 | 记录展示的管理员后台随机路径 | 🎉 安装完成 |

### 2.3 安装后验证

- [ ] `install.lock` 已生成
- [ ] `GET /api/v1/health` → `installed:true`
- [ ] `config.yaml` 含端口固化: `server.port=8080 / web.user_port=19876 / web.admin_port=19877`
- [ ] 访问 `/<admin随机路径>/` 管理后台可打开

---

## 3. 阶段三: 功能测试(前端全部按钮)

### 3.0 通用

- [ ] 注册页面: 提交注册 → 自动登录跳概览
- [ ] 登录页面: 错误密码提示 / 正确登录
- [ ] 右上角头像下拉: **修改密码**(弹窗: 错旧密拒绝/正确成功) → **退出登录**

### 3.1 用户中心

| 页面 | 按钮/操作 | 预期 |
|---|---|---|
| 概览 | (无按钮) | 统计卡片加载数值 |
| 发送消息 | 【发送】【重置】 | 发送后绿条 `受理成功 message_id=x`; 重置清空表单 |
| 消息记录 | 【查询】【重置】+ 搜索项(渠道/状态) | 列表按条件过滤; 分页/刷新/列设置正常 |
| API Key·Tab1 | 【新建 API Key】→ 提示明文(仅一次) / 【吊销】(确认弹窗) | 创建后列表出现; 吊销后状态禁用 |
| API Key·Tab2 Server酱 | 【新增 Key】→ 弹窗选协议/导入外部Key → 确定 / 【删除】/ 复制调用地址 | 新建/导入成功; 删除后立即失效 |
| 回调订阅 | 【注册回调】(URL/密钥/事件) / 【删除】 | 注册成功; 删除成功 |
| 套餐订阅 | 【立即购买】→ 弹窗【模拟支付】【跳转易支付】 | 模拟支付后订阅更新为当前套餐(按钮变"当前套餐") |
| 渠道配置 | 【配置渠道/修改配置】→ 表单保存 / 【重置为默认】(确认) | 保存后标签变"已配置"; 重置回默认 |
| 站内信 | 【全部已读】/ 行内【标记已读】 | 未读角标递减; 全部已读后角标消失 |
| 批量任务 | 【创建批量任务】→ 弹窗填收件人列表 → 确定 / 运行中【取消】 | 任务进度 100%; 取消后状态"已取消" |

### 3.2 管理后台

| 页面 | 按钮/操作 | 预期 |
|---|---|---|
| 平台概览 | (无按钮) | 统计卡片加载 |
| 租户管理 | 【禁用】(确认) →【启用】 | 状态标签切换 |
| 用户管理 | 【禁用】(确认) →【启用】 | 平台管理员不可禁用 |
| 模板审核 | 【批准】(确认) / 【驳回】→ 填原因弹窗 | 批准后移出待审; 驳回需原因 |
| 套餐管理 | 【新建套餐】→ 弹窗保存 / 【删除】 | 免费套餐不可删; 新建出现列表 |
| 支付订单 | 搜索/重置 | 列表加载(无数据则空态) |
| 审计日志 | 分页 | 记录含改密/审核等动作 |
| 系统设置 | 【保存 SMTP】【保存易支付】【保存】【轮换随机路径】(确认) | 保存提示成功; 轮换后新路径提示 |

---

## 4. 回归链路(端到端)

- [ ] 创建模板 → 提交审核 → 管理员批准 → 用模板发送(自动渠道路由)
- [ ] 发送站内信 → 状态 SUCCESS → 收件箱可见
- [ ] Server酱兼容: `curl -X POST --data-urlencode "title=test" "http://<域名>:8080/api/sctapi/<SendKey>.send"` → `{"code":0,...}`
- [ ] 幂等: 同 request_id 重发 → `duplicate:true`
- [ ] 限流: 管理后台额度调低 → 超限返回 429

---

## 5. 测试记录

| 日期 | 用例 | 结果(✅/❌) | 问题/备注 |
|---|---|---|---|
| | | | |
| | | | |

---

## 附录: 常用命令

```bash
# 健康检查
curl http://<域名>:8080/api/v1/health

# 云端部署最新代码
git push origin deploy && ssh mpcloud "sudo bash /opt/messagepusher-src/deploy/cloud-deploy.sh"

# 查看服务日志
ssh mpcloud "journalctl -u messagepusher -n 50 --no-pager"   # binary 模式
ssh mpcloud "docker compose -f /opt/messagepusher-docker/docker-compose.single.yml logs -f"   # compose 模式
```
