# Trigger 触发器

Trigger 用来把外部 Webhook 请求映射成内部模块动作，是 OctoManger 对外集成的标准入口。

## Trigger 能做什么

适用场景：

- 合作方推送用户注册事件
- 外部系统通知你执行某个账号动作
- 低代码地把“Webhook -> Job”串起来

限制：

- Trigger 只能绑定 `generic` 类型
- 最终执行的仍然是该类型对应的 OctoModule action

## 数据模型

Trigger 的核心字段：

| 字段 | 说明 |
| --- | --- |
| `name` | 展示名 |
| `slug` | Webhook 路径标识 |
| `type_key` | 目标账号类型 |
| `action_key` | 触发的动作 |
| `mode` | `async` 或 `sync` |
| `default_selector` | 默认账号筛选 |
| `default_params` | 默认参数 |
| `enabled` | 是否启用 |
| `token_prefix` | 触发 token 前缀 |

创建时会返回一次 `raw_token`，后续不会再返回。

## 访问地址

Webhook 地址固定是：

```text
/webhooks/{slug}
```

例如：

```text
https://localhost:8080/webhooks/github-register
```

## 鉴权方式

### 方式一：Trigger Bearer Token

```bash
curl -X POST https://localhost:8080/webhooks/github-register \
  -H "Authorization: Bearer <trigger-token>" \
  -H "Content-Type: application/json" \
  -d "{}"
```

### 方式二：Webhook API Key

```bash
curl -X POST https://localhost:8080/webhooks/github-register \
  -H "X-Api-Key: <webhook-key>" \
  -H "Content-Type: application/json" \
  -d "{}"
```

适用规则：

- `admin` Key 可触发全部 Trigger
- `webhook` Key 需要 `webhook_scope="*"` 或与当前 `slug` 匹配

## async 和 sync

### async

行为：

- 创建 Job
- 放入队列
- 立即返回 Job 元信息

适用：

- 执行耗时长
- 需要批量处理
- 调用方只关心是否成功入队

### sync

行为：

- 仍然创建 Job
- 但在 API 进程里直接执行
- 立即返回执行摘要

适用：

- 需要马上拿结果
- 账号数量较少
- 模块执行时间可控

## 请求体

可选请求体结构：

```json
{
  "mode": "async",
  "selector": {
    "identifiers": ["alice"]
  },
  "extra_params": {
    "force": true
  }
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `mode` | 可覆盖 Trigger 默认模式 |
| `selector` | 本次请求额外传入的筛选条件 |
| `extra_params` | 本次请求额外参数 |

## 参数合并规则

后端会做两次合并：

1. `default_selector + selector`
2. `default_params + extra_params`

后传入的字段会覆盖同名旧字段。

然后系统还会向最终 `params` 里注入 `_trigger` 元数据，例如：

```json
{
  "_trigger": {
    "endpoint_id": 1,
    "slug": "github-register",
    "type_key": "github",
    "action_key": "REGISTER",
    "mode": "async",
    "selector": {
      "identifiers": ["alice"]
    },
    "fired_at": "2026-03-10T12:00:00Z"
  }
}
```

模块可以读取它做审计、分流或幂等控制。

## 返回格式

### async 返回

通常包含：

- `endpoint`
- `mode`
- `queued=true`
- `input`
- `job`

### sync 返回

通常包含：

- `endpoint`
- `mode`
- `queued=false`
- `input`
- `job`
- `output`

`output` 里会带：

- `job_status`
- `matched_accounts`
- `processed_accounts`
- `results`

## 创建示例

```json
{
  "name": "GitHub Register",
  "slug": "github-register",
  "type_key": "github",
  "action_key": "REGISTER",
  "mode": "async",
  "default_selector": {
    "identifier_contains": "seed-"
  },
  "default_params": {
    "mode": "api"
  }
}
```

## 使用建议

- 外部系统只需要触发时，优先发放 `webhook` Key，不要发 `admin` Key
- 默认选择 `async`，把同步模式留给少量、低延迟动作
- 把通用参数沉到 `default_params`，让调用方只传差异字段
- 让模块消费 `_trigger` 元数据，便于追踪来源
