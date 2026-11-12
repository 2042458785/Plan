# 素序 SUXU · 精品零售系统

项目包含四个独立工程。

| 目录 | 用途 | 技术 |
|---|---|---|
| `backend/` | 商城 API | Go 1.26.2、Gin、Viper/YAML、Zap、GORM/MySQL、JWT、Redis 客户端 |
| `frontend/` | 顾客商城 | Vue 3、TypeScript、Vue Router 4、Vuex 4、Axios、Element Plus、Vite |
| `quick-frontend/` | 快速前台 | Vue 3、TypeScript、Vue Router 4、Vuex 4、Axios、Element Plus、Vite |
| `admin/` | 商家工作室 | Vue 2、Vue CLI 5、Vue Router 3、Vuex 3、Axios、Element UI |

顾客端和管理端分别安装、启动、构建、部署。顾客端不包含管理页面、管理入口或管理 API；后端独立校验管理令牌，不依赖隐藏入口提供权限保护。

## 后端配置与启动

首次使用时，将 `backend/src/config/config.example.yaml` 复制为同目录下的 `config.yaml`，再填写实际连接信息。也可以在 `config.local.yaml` 中填写本机覆盖值。两份运行配置均由 Git 忽略。

```yaml
mysql:
  host: "127.0.0.1"
  port: "3306"
  user: "你的MySQL用户名"
  password: "你的MySQL密码"
  dbname: "suxu_development"
  auto_create_database: true
jwt:
  secret: "至少32字符的随机字符串，请勿使用示例值"
bootstrap:
  admin_username: "admin"
  admin_password: "至少10字符的初始商家密码"
```

可以直接填写主配置，或按需创建本机覆盖文件。若 MySQL 账号被拒绝，请修改实际生效的配置；程序不会猜测密码、重置原库用户或回退到 JSON。

```sh
cd backend
go mod tidy
go run main.go
```

默认 API 端口为 **8090**，可在 `server.port` 修改。与参考工程一致，配置加载后启动 MySQL、执行 `AutoMigrate`、注入 DAO/Service/Controller，再启动路由。配置错误、连接失败或迁移失败会报错退出。

`mysql.auto_create_database: true` 时尝试创建配置中的独立数据库，账号需具备对应权限；为保护参考项目原库，程序只允许连接 `suxu_` 前缀数据库。设置为 `false` 时请先创建数据库，再使用具有该库建表/读写权限的账号启动。

首次启动在库内创建初始管理员（bcrypt 密码哈希），后续修改 YAML 初始密码不会覆盖已有管理员密码。

## 顾客端启动

```sh
cd frontend
npm install
npm run serve
```

默认 **8081**，同时输出 Local / Network 地址。顾客端参考项目的 `serve` 实际调用的就是 Vite，本工程保留同样的工具链。接口和静态资源统一配置在 `frontend/src/config.ts`；开发代理位于 `frontend/vite.config.ts`。

开发代理仅转发公开的商品、订单和图片 API。开发时 `/admin` 与管理 API 路径直接返回 404。

原版商城 `/` 的页面源码在 `frontend/src/views/StorefrontView.vue`。

## 快速前台启动

```sh
cd quick-frontend
npm install
npm run serve
```

快速前台入口为 `http://localhost:8083/quick/`，页面源码在 `quick-frontend/src/views/QuickStorefrontView.vue`，动态封面和分类在 `quick-frontend/src/components/QuickShop.vue`。在该目录执行 `npm run build`，产物独立写入 `quick-frontend/quick/`。两个前台通过各自的 API 请求读取同一个 Go 后端的商品和店铺设置，预约也进入同一个商家后台。部署在同一 IP 和端口时，顾客可在两个前台间继续使用购物袋。开发时两个端口各自保存购物袋。

## 独立商家端启动

```sh
cd admin
npm install
npm run serve
```

默认 **8082**，登录页 `/login`。使用 Vue CLI，配置在 `admin/vue.config.js`；接口地址等在 `admin/src/config.js`。

管理端使用 Bearer JWT。单端口部署时，前后台路径共享浏览器来源，路径不能隔离浏览器存储；顾客代码不包含管理功能或后台入口，权限仍由服务端校验。每次管理 API 请求校验签名算法、签发方、受众、角色、有效期、数据库会话和管理员状态；退出会撤销持久化会话。

各前端开发服务监听 `0.0.0.0`。同一 Wi-Fi 的手机使用终端输出的 Network IP 和对应端口。后端通过开发代理访问，可以继续只监听本机。

项目根目录的 `./start.sh` 可启动后端、原版商城和商家后台。快速前台在自己的目录运行 `npm run serve`。

后台「店铺设置」可上传或恢复快速小程序封面。

## 数据库结构

采用参考工程的 `sys_` 前缀与单数表名，持久化实际业务行，不再使用 `snapshots`：

| 表 | 数据 |
|---|---|
| `sys_category` | 商品分类 |
| `sys_product` | 商品资料与版本号 |
| `sys_product_sku` | 规格、颜色、独立库存与版本号 |
| `sys_product_image` | 后台上传的商品图片 |
| `sys_shop_setting` | 店铺信息、封面与预约开关 |
| `sys_order` | 预约、顾客资料、金额、状态、查询凭证和幂等信息 |
| `sys_order_item` | 下单时的商品、价格、规格、颜色快照 |
| `sys_stock_movement` | 初始库存、人工调整、预约扣减、取消返还流水 |
| `sys_admin_user` | 商家账号与密码哈希 |
| `sys_admin_session` | JWT 对应的持久化会话与撤销状态 |
| `sys_admin_audit_log` | 管理员变更记录 |
| `sys_migration_record` | 旧数据导入标记 |
| `sys_login_attempt` | 登录失败计数与临时封禁 |

下单、库存扣减和流水写入在同一个 MySQL 事务内完成。使用行锁、库存条件更新和幂等唯一约束防止并发超卖/重复下单；取消时在事务中返还库存。管理端编辑带版本号，旧页面不能覆盖新发生的库存或订单变更。金额始终使用整数分，不接受客户端定价。

Redis 客户端、配置及初始化入口保留。`redis.enable: true` 时必须能连接 Redis，并额外验证令牌缓存；默认关闭，商家会话仍通过 MySQL 持久化和撤销。扑克游戏的 WebSocket、链上交易、GPS、游戏 cron 不属于商城业务，未复制这些业务模块。

## 旧版 JSON 导入

如果本机有旧文件 `backend/data/boutique.json`，可以按下面的命令导入。导入必须在**商城库尚无商品时**执行，拒绝覆盖已有业务数据；整个导入在一个事务中完成，有导入标记防止重复执行。

```sh
cd backend
go run ./cmd/migrate-legacy -file data/boutique.json
```

导入后再运行 `go run main.go`。订单号、查询令牌、金额和状态保留。旧原型只记录商品总库存，无法恢复原本不存在的每个规格库存；导入时将总库存平均分配到规格组合，余数按顺序分配，总数保持一致。正式使用前应在商家端核实各 SKU 实物数量。

## 构建与检查

```sh
# 顾客端
cd frontend
npm run typecheck
npm run test:unit
npm run build
```

```sh
# 商家端
cd admin
npm run test:money
npm run build
```

```sh
# 后端：普通运行会明确标记 MySQL 集成用例 SKIP，不能当作通过
cd backend
go test -v ./...
go vet ./...

# 真实数据库验收，账号需能创建/删除测试专用 suxu_test_* 库
RUN_MYSQL_TESTS=1 go test -race -v -timeout 180s ./test/...
```

测试可通过 `SUXU_TEST_CONFIG=/绝对路径/config.yaml` 指向单独的验收 YAML，避免修改日常或线上配置；该变量仅供测试代码读取。2026-11-03 已用临时本地 MySQL 完成7组真实集成测试及浏览器下单/接单/重启验证，随后删除测试库和临时实例，详见 [验收记录](docs/mysql-acceptance.md)。

集成用例只创建并清理随机命名的 `suxu_test_*` 库，覆盖业务分表、SKU 库存、状态流、令牌/角色/会话撤销、重建应用后会话有效性、并发争抢、幂等、回滚、取消返还和旧版本覆盖拦截。数据库连接失败会 FAIL，不会悄悄替换为内存或 SQLite。

两个前端构建后，在根目录执行：

```sh
node scripts/check-architecture.mjs
```

该脚本检查前后台框架、必需架构文件、独立产物，以及顾客构建是否混入管理接口/令牌/入口。置信度与最终验证记录见 `docs/architecture-audit.md`。

## 独立部署

- 80 端口下的入口为完整商城 `/`、快速前台 `/quick/`、管理端 `/admin/login` 和接口 `/api/v1/...`。
- 完整 Nginx 配置：[docs/nginx.ip.conf](docs/nginx.ip.conf)，在一个server块内按路径分流；替换服务器 `/etc/nginx/nginx.conf`。
- 顾客构建产物：`frontend/quick/`，上传目录内容至 `/usr/share/nginx/html/dist/`。
- 快速前台构建产物：`quick-frontend/quick/`，上传目录内容至 `/usr/share/nginx/html/quick/`。
- 商家构建产物：`admin/dist_end/`，上传目录内容至 `/usr/share/nginx/html/dist_end/`。
- Go 后端只提供 API，默认不提供任何前端页面。
- 前端统一访问 `/api/v1`，由 Nginx 转发到本机 `127.0.0.1:8090`。
- 后台二维码默认指向当前服务器的80端口商城，可在弹窗中修改网址。
- 服务器安全组和防火墙只需允许 TCP 80，无须开放8080、8082或8090。生产部署由 Nginx 直接提供构建产物，无须运行前端开发服务。

完整步骤与检查命令见 [IP部署说明](docs/deployment-ip.md)。原域名部署示例仅作为备用。

当前仍为响应式 H5 商城；微信原生小程序和在线支付尚未接入。商品图与资料是可编辑演示配置，生成素材记录在 `docs/image-prompts.json`。本次重构保留已经制作的精品店视觉，不将“最好看”等主观目标当作已客观验证的事实。
