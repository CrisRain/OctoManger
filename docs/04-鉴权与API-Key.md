# 鉴权与 API Key

OctoManger 没有用户名密码体系，后台管理完全依赖 API Key。

## 基本规则

- 控制台和 API 共用同一套 `X-Api-Key` 认证方式
- 首个 Admin Key 只能在系统初始化时创建一次
- Webhook 既支持专属 Bearer Token，也支持 `role=webhook` 的 API Key
- 模块内部调用使用系统自动维护的 `internal` Key，用户无需手动管理

## 哪些接口不需要 Admin Key

公开接口只有这些：

- `GET /healthz`
- `GET /api/v1/system/status`
- `POST /api/v1/system/setup`

特殊说明：

- `POST /webhooks/{slug}` 不需要 Admin Key，但它本身仍然要求 Trigger Token 或 Webhook API Key
- `POST /api/v1/system/migrate` 需要 Admin Key

## 初始化流程

### 1. 查看是否需要初始化

```bash
curl http://localhost:8080/api/v1/system/status
```

返回：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "initialized": false,
    "needs_setup": true
  }
}
```

### 2. 创建首个 Admin Key

```bash
curl -X POST http://localhost:8080/api/v1/system/setup \
  -H "Content-Type: application/json" \
  -d "{\"admin_key_name\":\"Admin Key\"}"
```

返回值里的 `raw_key` 只会出现这一次。

### 3. 后续请求都带上 `X-Api-Key`

```bash
curl -H "X-Api-Key: <admin-key>" http://localhost:8080/api/v1/account-types/
```

## API Key 角色

### `admin`

拥有全部后台接口权限。

特点：

- 只能在系统初始化时创建
- 一旦系统里已经存在启用中的 Admin Key，就不能再创建新的 Admin Key

### `webhook`

用于外部系统触发 Trigger。

特点：

- 可以限制到单个 `slug`
- 不限制时 `webhook_scope="*"`
- 可用于请求 `/webhooks/{slug}`

### `internal`

系统自动生成，供 OctoModule 内部 API 使用。

特点：

- 存储在数据库系统配置中
- API 和 Worker 启动时会自动检查并补齐
- 不建议手工创建或复用

## 管理 API Key

接口：

- `GET /api/v1/api-keys/`
- `POST /api/v1/api-keys/`
- `PATCH /api/v1/api-keys/{id}`
- `DELETE /api/v1/api-keys/{id}`

创建 Webhook Key 示例：

```json
{
  "name": "partner-a",
  "role": "webhook",
  "webhook_scope": "partner-a-register"
}
```

返回时同样只会给一次 `raw_key`。

禁用 Key：

```json
{
  "enabled": false
}
```

## Trigger 的两种鉴权方式

### 方式一：Bearer Token

每个 Trigger 创建时都会生成独立 token。

请求示例：

```bash
curl -X POST https://localhost:8080/webhooks/demo \
  -H "Authorization: Bearer <trigger-token>" \
  -H "Content-Type: application/json" \
  -d "{}"
```

### 方式二：Webhook API Key

请求示例：

```bash
curl -X POST https://localhost:8080/webhooks/demo \
  -H "X-Api-Key: <webhook-key>" \
  -H "Content-Type: application/json" \
  -d "{}"
```

校验规则：

- `admin` 可以触发任意 Trigger
- `webhook` 只有 `webhook_scope="*"` 或与当前 `slug` 一致时才能触发

## 控制台登录

前端不会额外做账号体系登录，而是把 Admin Key 保存在浏览器本地，用它去请求后端。

这意味着：

- 换浏览器或清空站点存储后需要重新输入 Admin Key
- 只要 Key 失效、被删或被禁用，控制台就会重新要求登录

## 安全建议

- 把 `raw_key` 当作密码对待，只保存在密码管理器或安全变量里
- 生产环境建议上传正式证书，不要长期暴露自签名证书
- 对外只发放 `webhook` Key，不要把 `admin` Key 放进自动化脚本
- 模块内部如需访问后台，优先使用系统注入的 `context.api_token`，不要把管理 Key 硬编码到脚本里

## 常用接口示例

查看 API Key 列表：

```bash
curl -H "X-Api-Key: <admin-key>" http://localhost:8080/api/v1/api-keys/
```

创建 Webhook Key：

```bash
curl -X POST http://localhost:8080/api/v1/api-keys/ \
  -H "X-Api-Key: <admin-key>" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"partner-a\",\"role\":\"webhook\",\"webhook_scope\":\"partner-a-register\"}"
```
