# AGENTS.md: internal/models — 数据契约中心

## 定位
全部业务模型(GORM 标签)+ 状态/角色/事件常量单文件集中定义。**无 Tenant 模型**(2026-08 折叠入 User); User 无 name(合并为 nickname)。

## 核心语义

- **敏感字段一律 `json:"-"`**: User.PasswordHash/TotpSecret、Admin.PasswordHash/TotpSecret、APIKey.KeyHash、WebhookSubscription.Secret、PaymentOrder.NotifyData — 新增敏感字段必须照此, 否则明文泄漏到 API 响应。
- **复合唯一索引命名 `uk_<租户>_<key>`**: uk_tenant_code(模板)、uk_tenant_request(幂等)、uk_tenant_channel_target(退订)。
- **消息状态大写**(PENDING/SENDING/SUCCESS/FAILED/RETRYING/DEAD/CANCELLED)vs 业务状态小写(active/disabled/pending)— 勿混用。
- **Channel.TenantID=0 语义 = 平台默认**(区别于业务表 tenant_id=归属用户 ID)。
- **CompatSource 常量**: `serverchan_v1`(sc 形态, 显示名 **Server酱3**)/ `serverchan_v2`(sctapi 形态, 显示名 **Server酱·Turbo版**)— **存储值永不变**(DB 数据 + 路由匹配依赖), 显示名仅前端 ApiKeys.tsx 映射。

## 文件

- `models.go`: 全部模型 + 常量(状态/角色/事件类型/兼容源)
- 其余业务包引用本包常量, 勿在包内重写字面量(store 包例外: 消息状态直接写字符串, 见 store/AGENTS.md)

## 已踩坑/必须

- 租户已折叠: 新增"租户级"字段直接放 User, 不再建 Tenant 模型; 业务表 tenant_id 列名保留、值 = 归属用户 ID
- 模型改动直接反映到 AutoMigrate(无迁移文件), 删列需评估旧库残留(migrateLegacySchema 只清 tenant_id/name)
