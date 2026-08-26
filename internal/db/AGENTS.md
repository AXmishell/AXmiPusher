# AGENTS.md: internal/db — GORM 初始化与迁移

## 定位
三方言(PostgreSQL / MySQL 5.7 / SQLite)拨号器切换 + 16 表 AutoMigrate + 启动时旧库迁移。全仓唯一数据库入口(app.Build 经 db.Open)。

## 核心语义

- 方言切换: `cfg.Database.Type` → postgres/mysql/sqlite, DSN 由 `config.DSN()` 生成; 未知类型报错。
- **SQLite 单写连接**(`SetMaxOpenConns(1)`): 避免并发写锁冲突(store 认领依赖此串行性)。
- AutoMigrate: models 全部业务表 + `migrateLegacyAdmins` + `migrateLegacySchema`。
- **migrateLegacySchema**: 清理旧版 users 残留 `tenant_id`/`name` 列(租户折叠/名称合并前遗留); SQLite 删列前先 `DROP INDEX`, PG/MySQL 查 `information_schema.columns` 探测。
- **migrateLegacyAdmins**: users.role=platform_admin → admins(super_admin); **admins 表已有数据则只清 users 残留、绝不覆盖新数据**(事务逐行复制保留原时间戳)。

## 文件

- `db.go`: Open(拨号 + AutoMigrate + 单写连接)+ migrate + migrateLegacySchema + migrateLegacyAdmins

## 已踩坑/必须

- SQLite 删列前必须先删依赖该列的索引(否则 "error in index ... after drop column")
- 无版本化迁移(建表全走 AutoMigrate); migrations/ 为空目录, 生产 PG 建议补版本化迁移
- 新增表必须加进 migrate() 的 AutoMigrate 列表
