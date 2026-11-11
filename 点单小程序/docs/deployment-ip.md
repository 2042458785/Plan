# 只开放80端口的 IP 部署

下文的 `192.0.2.10` 是示例地址，访问时请换成实际公网 IP。

| 服务 | 地址 | 实际位置 |
|---|---|---|
| 原版完整商城 | `http://192.0.2.10/` | `/usr/share/nginx/html/dist/` |
| 快速前台 | `http://192.0.2.10/quick/` | `/usr/share/nginx/html/quick/` |
| 后台登录 | `http://192.0.2.10/admin/login` | `/usr/share/nginx/html/dist_end/` |
| 商品接口 | `http://192.0.2.10/api/v1/catalog` | Nginx 转发至 `127.0.0.1:8090` |

安全组只需开放80端口。前后台仍分别构建，不把管理页面或入口编入顾客包。两者共享同一个浏览器来源，真正的管理权限由Go后端JWT校验。

## 上传并应用

后台已经改为 `/admin/` 资源前缀和路由根路径，须重新上传新版构建，不能继续使用此前按8082根路径构建的旧包。

将新版 `suxu-web-quick.tar.gz` 和本项目 [nginx.ip.conf](nginx.ip.conf) 上传至服务器 `/root/`。本次需要应用带独立 `/quick/` 目录的新配置。

```sh
mkdir -p /usr/share/nginx/html
tar --no-same-owner -xzf /root/suxu-web-quick.tar.gz -C /usr/share/nginx/html
cp /etc/nginx/nginx.conf /etc/nginx/nginx.conf.bak
cp /root/nginx.ip.conf /etc/nginx/nginx.conf
nginx -t && systemctl reload nginx
```

三个静态目录分别包含自己的 `index.html`。Nginx 配置有四个 `server` 块，前台商城监听公网 80，后端接口、快速商城和后台前端分别监听本机 `127.0.0.1:18091`、`:18092`、`:18093`。公网入口按路径转发，后端接口块再把请求转发给 Go 的 `127.0.0.1:8090`。

自行构建时，在本项目根目录运行：

```sh
npm --prefix frontend run build
npm --prefix quick-frontend run build
npm --prefix admin run build
```

分类、商品图片和配送订单需要同时部署新版 Go 后端与三个前端构建。后端启动时迁移图片表和订单字段。新库使用 `bootstrap.seed_catalog: false`，后台在「商品与库存」先添加分类，再上传商品，在「店铺设置」开启预约。已有数据由后台管理。图片上传支持最多 4 MB，Nginx 配置已包含对应的请求大小限制。

各项目可单独构建及上传。分别将 `frontend/quick/`、`quick-frontend/quick/`、`admin/dist_end/` 的内容放入上表对应目录。开发端口分别是 8081、8083、8082。

## 检查

在服务器执行：

```sh
curl -I http://127.0.0.1/
curl -I http://127.0.0.1/quick/
curl -I http://127.0.0.1/admin/login
curl -i http://127.0.0.1/api/v1/catalog
curl -i http://127.0.0.1/api/v1/admin/session
```

三个页面入口应返回 200 及各自的 HTML。商品接口应返回 JSON。未登录的商家会话接口应返回 401。

2026-11-04，四个 `server` 块已通过本机 Nginx 1.26.3 语法检查。通过同一个临时入口验证了三个前端页面、快速商城静态资源，以及 API GET 和 POST 转发。API 测试使用临时响应服务，验证了路径、查询参数、请求头和请求体。

尚未连接服务器应用新配置。实际服务器仍需执行 `nginx -t` 并重载。资源前缀及路由根路径的设置分别遵循 [Vue CLI publicPath](https://cli.vuejs.org/config/#publicpath) 与 [Vue Router base](https://v3.router.vuejs.org/api/#base)。
