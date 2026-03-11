# OctoManger

OctoManger 是一个面向多账号运营场景的控制台与执行平台。它把账号类型、账号记录、异步任务、Webhook Trigger、邮箱能力和 Python OctoModule 统一到同一套后台里，适合做批量验证、资料拉取、注册编排、守护监控等自动化流程。

## 核心能力

- `generic` 类型账号可绑定 Python OctoModule，按 `action` 执行业务逻辑。
- `Job + Worker + JobRun` 负责批量执行、失败记录、重试和取消。
- `Trigger` 可把外部 Webhook 映射成同步或异步任务。
- 内置 Outlook 邮箱账号管理，支持 OAuth 手动接入、Graph Token 批量导入、邮件读取。
- 控制台可直接查看和编辑模块脚本、模块目录、依赖环境与运行历史。
- API Key、系统配置、自签名 TLS、证书上传都在后台内统一管理。

## 快速体验

```bash
docker compose up -d --build
```

如果你要跑依赖 Playwright 的 OctoModule，且主应用在 Docker 里运行，推荐把 OctoModule 服务切到宿主机：

```bash
# 1. 在项目根目录创建/修改 .env
# OCTOMODULE_SERVICE_URL=http://host.docker.internal:8091
# OCTOMODULE_SERVICE_EMBEDDED=false

# 2. 启动容器内主应用
docker compose up -d --build

# 3. 在宿主机单独启动 octomodule service
cd backend
$env:PATHS_OCTO_MODULE_DIR="..\scripts\python\modules"
$env:PYTHON_BIN="python"
go run ./cmd/octomodule
```

这样 API / Worker 继续留在容器里，Playwright 则在宿主机浏览器环境执行，稳定性会明显好于容器内运行。

如果你仍然要在容器内安装 Chromium，建议同时在 `.env` 里加上下载镜像或代理配置：

```env
# 可替换成你自己的内网制品库 / CDN / 镜像
PLAYWRIGHT_DOWNLOAD_HOST=https://<your-playwright-mirror>
PLAYWRIGHT_DOWNLOAD_CONNECTION_TIMEOUT=120000
PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
# 如果需要代理，再补这些
# HTTP_PROXY=http://host.docker.internal:7890
# HTTPS_PROXY=http://host.docker.internal:7890
```

`docker-compose.yml` 已经把这些环境变量透传给 `app` 容器，并持久化了浏览器缓存目录，避免每次重新下载 Chromium。

默认访问地址：

- `https://localhost:8080`
- `http://localhost:8080` 也可以，会被同端口自动重定向到 HTTPS

首次启动时，如果数据库里还没有证书，服务会自动生成一张自签名证书。浏览器出现证书告警属于预期行为。

## 文档导航

| 文档 | 说明 |
| --- | --- |
| [01-快速开始](docs/01-快速开始.md) | 从零启动、初始化后台、跑通第一条任务 |
| [02-架构说明](docs/02-架构说明.md) | 核心组件、数据流、响应约定 |
| [03-部署与配置](docs/03-部署与配置.md) | Docker、本地开发、配置项、TLS 行为 |
| [04-鉴权与API-Key](docs/04-鉴权与API-Key.md) | 初始化、Admin Key、Webhook Key、鉴权边界 |
| [05-账号类型与账号](docs/05-账号类型与账号.md) | Account Type、Account、脚手架和状态语义 |
| [06-OctoModule开发指南](docs/06-OctoModule开发指南.md) | 模块输入输出协议、`octo.py`、依赖与 daemon |
| [07-任务系统](docs/07-任务系统.md) | Job、selector、调度、JobRun、批量任务 |
| [08-Trigger触发器](docs/08-Trigger触发器.md) | Webhook 接入、同步/异步模式、参数合并 |
| [09-邮箱账号](docs/09-邮箱账号.md) | Outlook OAuth、Graph 导入、邮件读取、批量生成记录 |
| [10-系统设置与SSL](docs/10-系统设置与SSL.md) | 系统配置、健康检查、证书上传与本地 TLS |

开发相关补充：

- [GitHub OctoModule](scripts/python/modules/github/README.md)
- [邮箱批量注册内部说明](docs/dev/邮箱批量注册.md)

## 仓库结构

```text
.
├── backend/                 Go API、Worker、Scheduler、Daemon
├── web/                     Bun + Vite + React 控制台
├── scripts/python/modules/  OctoModule 脚本与共享 octo.py
├── configs/                 默认配置文件
├── docs/                    用户文档与内部文档
├── docker-compose.yml
└── Dockerfile
```

## 本地开发

先启动依赖：

```bash
docker compose up -d postgres redis
```

再分别启动后端和前端：

```bash
cd backend
SERVER_TLS=false go run ./cmd/octomanger
```

```bash
cd web
bun dev
```

本地前端开发默认走 Vite 代理到 `http://localhost:8080`，所以调试时建议把后端的 `SERVER_TLS` 设为 `false`。更多命令见 [03-部署与配置](docs/03-部署与配置.md)。
