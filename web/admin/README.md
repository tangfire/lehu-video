# 校园 e站运营后台

这是校园 e站的运营后台前端，已并入后端主项目。默认入口是 `/admin`，不再包含旧短视频站、聊天、好友、群组等用户侧页面。

## 本地 Docker 启动

先启动后端：

```bash
cd /Users/firetang/Documents/lehu/lehu-campus
docker compose up -d --build
```

单独启动运营后台：

```bash
cd /Users/firetang/Documents/lehu/lehu-campus/web/admin
docker compose up -d --build
```

访问地址：

```text
http://localhost:15173/admin
```

后端默认：

```text
http://localhost:18080/v1
```
