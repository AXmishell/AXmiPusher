# AGENTS.md: internal/config — 配置加载与持久化

## 定位
三层配置源(MP_* env > config.yaml > 默认值)+ 安装锁门控 + 后台路径校验/随机生成 + 三方言 DSN。全仓唯一配置契约中心。

## 核心语义

- **config.yaml 路径硬编码 CWD**(`const FilePath = "config.yaml"`), 无 env 覆盖; 文件缺失**不视为错误**(Load 返回默认配置), 业务 API 凭 install.lock 判定"未安装"返回 503。
- 配置优先级: `MP_*` 环境变量 > config.yaml > 默认值(ParseConfig: 反序列化 → applyEnv → applyDefaults); Save 由安装向导/轮换接口调用。
- **DSN 三方言**(`Config.DSN()`):
  - sqlite: 裸路径(SQLitePath); postgres: key=value DSN(host/port/user/password/dbname/sslmode)
  - mysql: `user:pass@tcp(host:port)/db?charset=utf8mb4&parseTime=True&loc=Local`(5.7 兼容), 端口缺省 3306
- **ValidateAdminPath(p)**: 去首尾斜杠 → `^[A-Za-z0-9]{8,32}$`; 保留字 `api/install/assets/login/register` 禁用(与路由树/SPA 路由冲突); 返回规范化路径。
- **GenerateRandomAdminPath()**: crypto/rand + 62 字符集(a-z/A-Z/0-9)生成 **16 位**; 安装向导与随机轮换共用(熵高于 hex, 满足校验)。

## 文件

- `config.go`: Config 结构(10 段)+ Load/ParseConfig/Save + IsInstalled/MarkInstalled + applyEnv(24 个 MP_*)+ DSN + ValidateAdminPath/GenerateRandomAdminPath
- `config_test.go`: 旧字段向后兼容 + TestDSN_MySQL + TestValidateAdminPath(表驱动)+ TestGenerateRandomAdminPath

## 已踩坑/必须

- config.yaml 含 JWT 密钥/DB 密码: **绝不提交**(config.yaml.bak 曾泄露历史密钥)
- 改 DSN/校验规则先看 config_test.go 断言(TestDSN_MySQL 锁死 DSN 形状, TestValidateAdminPath 锁死 8-32 位/保留字)
- 后台路径规则变更需同步: config.go + install.html(pattern)+ web/admin Settings.tsx(placeholder)
