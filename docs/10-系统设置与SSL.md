# 系统设置与 SSL

本页对应控制台里的 `Settings` 和 `SSL` 两个入口，主要负责健康检查、系统配置项保存，以及 API 服务的证书管理。

## 健康检查

公开接口：

- `GET /healthz`

响应格式：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "status": "ok",
    "time": "2026-03-10T12:00:00Z"
  }
}
```

这个接口不需要 API Key，适合反向代理、容器探针和运维巡检使用。

## 系统配置

统一接口：

- `GET /api/v1/config/{key}`
- `PUT /api/v1/config/{key}`

请求体固定是 JSON：

```json
{
  "value": "OctoManger"
}
```

当前控制台内置展示的键：

| Key | 说明 |
| --- | --- |
| `app.name` | 应用展示名称 |
| `job.default_timeout_minutes` | 默认任务超时分钟数 |
| `job.max_concurrency` | Worker 最大并发数的配置记录 |
| `outlook_oauth_config` | Outlook OAuth 配置，邮箱页面直接使用 |

注意事项：

- 系统配置值保存在数据库 `system_config` 表中。
- `outlook_oauth_config` 已经接入邮箱页面流程。
- 其余键目前主要用于配置持久化和界面展示，不代表运行时一定会自动读取并生效。

## 默认 TLS 行为

`server.tls` 默认是开启的。

当 API 以 TLS 模式启动时：

- 服务监听一个端口，同时接收 HTTPS 和明文 HTTP。
- 明文 HTTP 请求会被自动 301 到同端口的 HTTPS 地址。
- 如果数据库中还没有证书和私钥，服务会自动生成一张自签名证书。

自动生成证书的特征：

- ECDSA P-256
- 有效期 10 年
- SAN 包含 `localhost`、`127.0.0.1`、`::1`

这意味着：

- Docker Compose 默认可以直接用 `https://localhost:8080`
- 浏览器首次访问会提示证书不受信任，这属于预期行为

## 上传自定义证书

控制台对应接口：

- `GET /api/v1/ssl/certificate`
- `PUT /api/v1/ssl/certificate`
- `DELETE /api/v1/ssl/certificate`

保存时需要同时提交 PEM 证书和 PEM 私钥：

```json
{
  "cert": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
  "key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----"
}
```

返回值包含：

- 当前证书原文
- 是否已保存私钥
- 解析后的 `subject`、`issuer`、`not_before`、`not_after`、`sans`

私钥不会通过读取接口返回。

## 本地开发建议

如果你在 `web/` 里使用 `bun dev`，Vite 默认代理到 `http://localhost:8080`。这时有两种做法：

1. 推荐：启动后端时设置 `SERVER_TLS=false`
2. 或者：把前端代理改成 HTTPS，并处理自签名证书问题

Outlook OAuth 回调地址也要和当前页面同源：

- 后端静态托管时通常是 `https://localhost:8080/oauth/callback`
- Vite 开发模式下通常应改成 `http://localhost:5173/oauth/callback`

## 常用操作

查看健康状态：

```bash
curl https://localhost:8080/healthz
```

读取系统配置：

```bash
curl -H "X-Api-Key: <admin-key>" https://localhost:8080/api/v1/config/app.name
```

写入系统配置：

```bash
curl -X PUT \
  -H "X-Api-Key: <admin-key>" \
  -H "Content-Type: application/json" \
  -d "{\"value\":\"OctoManger\"}" \
  https://localhost:8080/api/v1/config/app.name
```
