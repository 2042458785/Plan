# 本地 MySQL 验收记录

验收日期：2026-11-03（北京时间）。结论：以下本地验收通过，测试库及临时实例已删除。

## 环境与范围

本机 MySQL 9.3.0，使用独立数据目录与 `127.0.0.1:13306`。临时 YAML 与日常配置分开，未连接线上数据库，未改动原有3306实例及本项目私有配置。

后端使用实际 Gin 路由、Service、DAO、GORM 和 MySQL/InnoDB。自动化接口测试经 `httptest` 调用路由；浏览器联调通过实际8081/8082开发服务和8090后端网络完成。没有以模拟数据库或SQLite代替MySQL。

## 自动化结果

实际命令（配置路径用占位表示，临时文件已清理）：

```sh
RUN_MYSQL_TESTS=1 SUXU_TEST_CONFIG=<临时YAML绝对路径> go test -race -v -timeout 180s ./test/...
```

最终结果：`PASS`，`Boutique/test 14.167s`，无竞态报告；以下7组全部通过，未跳过：

| 用例 | 验证内容 |
|---|---|
| BusinessTablesAndSKUSeed | 自动创建12张业务表、7件商品、戒围分别对应独立SKU |
| LifecycleAuthorizationAndRevocation | 下单、查询凭证、完整状态流、拒绝非法倒退、重建应用后会话保留、退出撤销 |
| MerchantCatalogSettingsAndDatabaseQueries | 商品新建与编辑、金额分准确存储、店铺开关与版本检查、订单筛选分页、营业汇总、操作审计 |
| LegacyImportAtomicityAndRepeatProtection | 金额错误导入回滚、正确导入保留历史快照与凭证、重复导入拒绝 |
| ForgeryExpiryRoleAndRateLimit | 伪造签名、过期、错误角色拒绝，连续错误密码触发封禁 |
| ConcurrentInventoryAndIdempotency | 12个请求抢3件库存恰好3单成功；8个相同幂等请求只生成1单、扣库存1次 |
| RollbackCancellationAndStaleEdits | 多商品失败整单回滚、取消仅返还1次、旧商品版本拒绝覆盖、同幂等键不同内容拒绝 |

另执行修复后的 `go test ./...`、`go vet ./...` 和顾客构建产物隔离检查，均通过。普通测试默认跳过真实数据库用例，与上面显式启用的验收结果分开记录。

## 发现与修复

1. 同一分类有多件商品时，初始化查询误把商品排列位置也当作分类匹配条件，导致唯一名称重复。已改为按名称查找，位置只作为创建属性。
2. 测试断言复用已填充主键的GORM目标结构，查询另一SKU时额外叠加旧主键。已清空目标结构并检查查询错误，随后重新验证库存和取消结果。

## 浏览器与 SQL 核对

顾客在8081商城选择莫桑钻戒“港码12、银白色、1件”，提交合成测试预约；商家在独立8082后台登录后看到相同订单，并依次确认、备货完成。

| 核对项 | 结果 |
|---|---|
| 金额 | 69900分 / ¥699.00 |
| 状态与版本 | ready（待自提），版本3 |
| 所选规格余量 | 港码12余2件，其他戒围各3件 |
| 实际后端进程重启 | 后台会话仍有效，订单仍为待自提 |
| 顾客刷新后查询 | 从原浏览器预约记录读到同号、同规格和待自提状态 |
| 商家退出 | 返回独立登录页 |

## 清理结果

自动化创建的随机 `suxu_test_*` 库已由用例清理。手动联调的 `suxu_acceptance` 已执行删除，随后查询：

```text
remaining_test_databases
0
Database
information_schema
mysql
performance_schema
sys
```

临时8090后端与13306 MySQL进程均正常退出。临时数据目录、测试YAML、日志及套接字已删除；端口复核只剩原有3306 MySQL监听。本项目日常私有配置修改时间未变化。

保留的仅为修复后的代码、可重复运行的测试、此记录。线上数据库连接信息由用户自行填写，未在本次验收。Redis启用路径也不包含在此次结果中。

结构化结果见 [mysql-acceptance.json](test-results/mysql-acceptance.json)。
