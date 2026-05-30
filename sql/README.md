# SQL 使用说明

## 全新生产库

首发生产环境会使用全新的云 MySQL，所以只需要执行合并后的初始化脚本：

```bash
mysql -h <云 MySQL 内网地址> -u <业务账号> -p < sql/campus.sql
```

`campus.sql` 会创建 `lehu_campus_db`，并一次性创建当前项目需要的账号、用户、社区、审核、通知、e仔/RAG、Agent、运维日志等表和默认运营配置。

执行完 `campus.sql` 后，不要再把 `2026*.sql` 从头到尾重复执行一遍。历史增量脚本里的结构已经折叠进 `campus.sql`。

## 历史增量脚本

`2026*.sql` 文件保留给两类场景：

- 已经存在的本地开发库或旧测试库，需要补某一轮新增字段、索引或默认配置。
- 首发上线之后，线上库已经有数据，后续版本需要做向前兼容变更。

以后新增数据库结构时，规则是：

- 同步更新 `campus.sql`，保证全新安装永远只有一个入口。
- 新增一个时间戳命名的增量 SQL，给已有库升级使用。
- 增量 SQL 优先只做新增表、字段、索引和默认配置，不做破坏性删除。

## 上线前检查

初始化前确认：

- 云 MySQL 与应用服务器同地域，优先走内网地址。
- 3306 不开放公网，只允许应用服务器安全组访问。
- `.env.production` 里的 `LEHU_MYSQL_DSN` 指向同一个库名 `lehu_campus_db`。

初始化后确认：

```sql
SHOW DATABASES LIKE 'lehu_campus_db';
USE lehu_campus_db;
SHOW TABLES;
SELECT setting_key, setting_value FROM campus_ops_setting;
```
