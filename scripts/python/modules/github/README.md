# GitHub OctoModule

这是一个面向 GitHub 账号的 OctoModule，提供两类能力：

- 对已有 GitHub 账号执行 API 操作
- 通过邮箱验证码和 2captcha 尝试自动注册新账号

## 已实现动作

| Action | 说明 |
| --- | --- |
| `VERIFY` | 校验 Token 是否可用 |
| `HEALTH_CHECK` | `VERIFY` 的别名 |
| `GET_PROFILE` | 获取当前认证用户资料 |
| `LIST_REPOSITORIES` | 列仓库 |
| `GET_REPOSITORY` | 取单个仓库详情 |
| `CREATE_REPOSITORY` | 创建仓库 |
| `CREATE_ISSUE` | 创建 Issue |
| `REGISTER` | 自动注册 GitHub 账号 |

入口文件：

```text
scripts/python/modules/github/main.py
```

## 文件结构

```text
github/
├── main.py
├── client.py
├── register.py
├── captcha.py
├── account_type.github.json
├── requirements.txt
└── README.md
```

## 导入账号类型

推荐直接使用仓库里的类型模板：

```text
scripts/python/modules/github/account_type.github.json
```

它定义了：

- `key = github`
- `category = generic`
- GitHub 账号 `schema`
- `REGISTER` 所需参数

## 账号 `spec`

已有账号常用字段：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `username` | 是 | GitHub 用户名 |
| `token` | 是 | Personal Access Token |
| `api_base_url` | 否 | 默认 `https://api.github.com` |
| `user_agent` | 否 | 自定义 UA |
| `timeout_seconds` | 否 | 默认 30，范围 5 到 120 |
| `default_owner` | 否 | 不填时回退到 `identifier` |

说明：

- 运行 API 类动作时，`token` 必须可用
- 执行 `REGISTER` 时，账号记录可以先只存 `username` 和 `password`，后续再回填 `token`

## 依赖

默认模式是 `api`，不依赖 Playwright。

只有使用 `browser` 注册模式时，才需要：

```bash
pip install playwright
python -m playwright install chromium
```

`requirements.txt` 目前只声明了 `playwright>=1.44.0`，也是为 `browser` 模式准备的。

## 对已有账号的常用操作

### VERIFY

最小账号：

```json
{
  "type_key": "github",
  "identifier": "myuser",
  "status": 1,
  "spec": {
    "username": "myuser",
    "token": "ghp_xxx"
  }
}
```

Job 示例：

```json
{
  "type_key": "github",
  "action_key": "VERIFY",
  "selector": {
    "identifiers": ["myuser"]
  },
  "params": {}
}
```

### CREATE_REPOSITORY

```json
{
  "name": "my-new-repo",
  "description": "Created via OctoManger",
  "private": false,
  "auto_init": true
}
```

### CREATE_ISSUE

```json
{
  "repo": "my-new-repo",
  "title": "First issue",
  "body": "Issue body here"
}
```

## REGISTER 依赖的外部条件

执行 `REGISTER` 前至少要准备：

- 一个可用的邮箱账号记录，用于收 GitHub 验证码
- 一个 2captcha Key
- 可选的代理

当前模块通过参数里的：

- `email_api_url`
- `email_api_key`
- `email_account_id`

去拉取 OctoManger 内部邮箱接口，并轮询最新邮件。

## REGISTER 参数

必填参数：

| 参数 | 说明 |
| --- | --- |
| `username` | 目标 GitHub 用户名 |
| `password` | 登录密码 |
| `email` | 注册邮箱地址 |
| `email_account_id` | OctoManger 邮箱账号 ID |
| `email_api_url` | OctoManger 后端地址 |
| `email_api_key` | OctoManger Admin API Key |
| `twocaptcha_api_key` | 2captcha API Key |

常用可选参数：

| 参数 | 说明 |
| --- | --- |
| `mode` | `api` 或 `browser`，默认 `api` |
| `email_mailbox` | 监听邮箱目录，默认 `INBOX` |
| `wait_seconds` | 最长等待邮件秒数 |
| `poll_interval` | 轮询间隔秒数 |
| `proxy` | HTTP / HTTPS 代理 |
| `headless` | `browser` 模式下是否无头 |
| `arkose_public_key` | 手动指定 Arkose 公钥 |
| `arkose_subdomain` | Arkose JS 子域 |

## `api` 与 `browser` 模式

### `api`

特点：

- 默认模式
- 不依赖 Playwright
- 直接请求 GitHub 注册接口
- 需要依赖传入或兜底的 Arkose 公钥

### `browser`

特点：

- 依赖 Playwright
- 在真实浏览器页面中完成注册
- 更适合页面结构频繁变化时排查
- 可通过 `headless=false` 观察页面过程

## 返回结果

模块遵循 OctoModule 标准输出，成功时会返回：

- `status = success`
- `result.event`
- 业务字段

失败时会返回：

- `status = error`
- `error_code`
- `error_message`

Worker 会把这些内容写入 `job_runs`。

## 调试建议

- 先对已有账号执行 `VERIFY`，确认 Token 和网络没问题
- `REGISTER` 优先先用 `mode=api`，失败再切 `mode=browser`
- 如果验证码总失败，手动传 `arkose_public_key`
- 观察控制台里的模块运行历史和 `job_runs.logs`

## 相关文档

- [OctoModule 开发指南](../../../docs/06-OctoModule开发指南.md)
- [邮箱账号](../../../docs/09-邮箱账号.md)
