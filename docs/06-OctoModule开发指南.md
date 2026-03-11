# OctoModule 开发指南

OctoModule 是 OctoManger 对 `generic` 账号类型的执行单元。一个模块本质上就是一个 Python 脚本，它读取 JSON 请求，执行指定 `action`，再把结果写回标准输出。

## 适用场景

适合放进 OctoModule 的逻辑：

- 账号注册
- 账号验证
- 拉取资料
- 调用第三方 API
- 登录后拿回 cookie / token
- 守护型监听任务

不适合直接塞进模块的逻辑：

- 大量状态共享的复杂编排
- 长事务数据库处理
- 需要强并发控制的核心调度

这类逻辑更适合留在后端服务层。

## 目录与入口

默认情况下，一个 `generic` 类型会对应一个目录：

```text
scripts/python/modules/<type_key>/
```

默认入口文件：

```text
scripts/python/modules/<type_key>/main.py
```

如果 `script_config` 指定了入口，则会改成对应相对路径。入口路径必须满足：

- 只能是相对路径
- 不能逃出 `paths.octo_module_dir`
- 不能写成绝对路径

## 脚手架文件

新建 `generic` 类型时，后端会自动生成最小脚手架，包含：

- `handle_register`
- `handle_verify`
- `ACTIONS` 分发表
- `octo.run_module(execute)`

你可以直接在这个脚手架上替换业务逻辑。

## 输入协议

Worker / Dry Run 传给模块的输入结构如下：

```json
{
  "action": "VERIFY",
  "account": {
    "identifier": "alice",
    "spec": {
      "token": "xxx"
    }
  },
  "params": {},
  "context": {
    "request_id": "1:2",
    "protocol": "ndjson.v1",
    "api_url": "http://127.0.0.1:8080",
    "api_token": "..."
  }
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `action` | 当前要执行的动作名 |
| `account.identifier` | 账号标识 |
| `account.spec` | 账号配置 |
| `params` | Job / Trigger / Dry Run 传入的参数 |
| `context.request_id` | 请求跟踪 ID |
| `context.protocol` | 当前桥接协议，通常是 `ndjson.v1` |
| `context.api_url` | 内部 API 地址 |
| `context.api_token` | 内部 API Token |

## 输出协议

模块最终必须返回一个 JSON 对象。

成功：

```json
{
  "status": "success",
  "result": {
    "event": "verified"
  }
}
```

失败：

```json
{
  "status": "error",
  "error_code": "VALIDATION_FAILED",
  "error_message": "token is required"
}
```

可选会话：

```json
{
  "status": "success",
  "result": {},
  "session": {
    "type": "cookie",
    "payload": {
      "cookie": "..."
    },
    "expires_at": "2026-03-11T00:00:00Z"
  }
}
```

## 实时日志

模块运行期间可以输出日志事件：

```json
{
  "status": "log",
  "level": "info",
  "message": "starting verify"
}
```

这些日志会被桥接层实时收集，并最终进入：

- 控制台运行历史
- `job_runs.logs`
- Worker 日志

## 推荐写法

最小模块通常长这样：

```python
import octo


def execute(request: dict) -> dict:
    action = str(request.get("action", "")).strip().upper()
    account = request.get("account") or {}
    identifier = str(account.get("identifier", "")).strip()
    spec = account.get("spec") or {}
    params = request.get("params") or {}

    if not identifier:
        return octo.error("VALIDATION_FAILED", "account.identifier is required")

    if action == "VERIFY":
        octo.emit_log("running verify", level="info", identifier=identifier)
        return octo.success({"event": "verified", "identifier": identifier})

    return octo.error("UNSUPPORTED_ACTION", f"unsupported action: {action}")


if __name__ == "__main__":
    raise SystemExit(octo.run_module(execute))
```

## `octo.py` 能做什么

共享 SDK 在：

```text
scripts/python/modules/octo.py
```

常用能力：

| 能力 | 说明 |
| --- | --- |
| `octo.success()` | 构造成功响应 |
| `octo.error()` | 构造错误响应 |
| `octo.emit_log()` | 输出结构化运行日志 |
| `octo.from_context()` | 从 `context` 构造内部 API 客户端 |
| `client.get_account()` | 读取账号详情 |
| `client.get_account_by_identifier()` | 按 `(type_key, identifier)` 查询账号 |
| `client.patch_account_spec()` | 回写账号 `spec` |
| `client.get_latest_email()` | 获取邮箱最新邮件 |

推荐写法：

```python
ctx = request.get("context") or {}
client = octo.from_context(ctx)
if client:
    account = client.get_account_by_identifier("github", "alice")
```

## Dry Run

开发模块时，先用 Dry Run，而不是直接建正式 Job。

控制台路径：

- `Modules -> <type_key> -> Dry Run`

接口：

```json
POST /api/v1/octo-modules/<type_key>:dry-run
{
  "action": "VERIFY",
  "account": {
    "identifier": "alice",
    "spec": {}
  },
  "params": {}
}
```

Dry Run 不落正式 Job，只返回模块执行结果。

## 正式执行

正式任务走 `Job`：

1. 创建 Job
2. Worker 读取账号集合
3. 按 selector 过滤
4. 逐个执行模块
5. 结果写入 `job_runs`

区别：

- Dry Run 只测试一条输入
- Job 面向真实账号集合

## 本地命令行测试

脚本支持从 stdin 读取一条 JSON。

示例：

```bash
echo '{"action":"VERIFY","account":{"identifier":"alice","spec":{}},"params":{},"context":{"request_id":"local"}}' | python scripts/python/modules/demo/main.py
```

如果脚本用到了 `octo.py`，确保 `scripts/python/modules/` 在 `PYTHONPATH` 中，或者直接在模块目录里运行。

## 依赖管理

每个模块目录都可以单独维护自己的虚拟环境：

```text
scripts/python/modules/<type_key>/.venv/
```

控制台支持：

- 查看虚拟环境状态
- 读取 / 编辑 `requirements.txt`
- 从 `requirements.txt` 安装依赖
- 直接安装额外包

手动安装示例：

```bash
cd scripts/python/modules/github
python -m venv .venv
.venv/Scripts/python.exe -m pip install -r requirements.txt
```

在 Linux / macOS 上把 `Scripts` 换成 `bin`。

## 运行历史

与模块相关的执行结果可以从两处查看：

- `Jobs` 页查看单个 Job 最近一次运行
- `Modules -> 运行历史` 查看按 `type_key` 聚合的 `job_runs`

`job_runs` 里会保存：

- 账号 ID
- 日志
- 成功结果
- 错误码与错误消息
- 开始与结束时间

## daemon 模式

如果 `capabilities` 里包含：

```json
{
  "daemon": {
    "action": "WATCH"
  }
}
```

Daemon Manager 会为该类型下 `status = 1` 的账号启动长驻子进程。

daemon 模块输出约定：

- `{"status":"init_ok"}`：初始化成功
- `{"status":"event","result":{...}}`：产生一条事件
- `{"status":"done"}`：主动退出
- `{"status":"error",...}`：致命错误
- `{"status":"log",...}`：运行日志

收到 `event` 后，Daemon Manager 会把它写成一条 `job_run`。

## 常见问题

### 模块脚本能执行，但控制台里看不到结果

优先检查：

- 是否返回了合法 JSON 对象
- 最终是否真的返回 `status=success` 或 `status=error`
- 是否把普通文本误写到了 stdout，而不是 `status=log`

### 模块导入第三方包失败

优先检查：

- 模块目录下是否已创建 `.venv`
- `requirements.txt` 是否在模块目录根部
- 运行时是否真的使用了该目录的虚拟环境

### 内部 API 调用 401

先确认：

- 模块是通过 OctoManger 调起的，不是你手动裸跑
- 输入里的 `context.api_url` 和 `context.api_token` 是否存在

### daemon 一直没启动

通常是以下原因：

- `capabilities.daemon` 没配置
- 账号 `status` 不是 `1`
- 模块入口文件不存在
