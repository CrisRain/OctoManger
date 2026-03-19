# OctoManger

OctoManger 是一个多账号自动化任务管理平台，支持通过 Python 插件扩展对任意类型账号（GitHub、邮箱等）执行自动化操作。

## 核心功能

| 功能 | 说明 |
|------|------|
| **账号管理** | 统一管理多种类型账号，字段 Schema 由插件自定义 |
| **任务系统（Jobs）** | Cron 定时调度与手动触发，记录完整执行历史与实时日志 |
| **Agent 监管** | 长驻后台的自动化 Agent，支持 SSE 实时日志流 |
| **触发器（Triggers）** | Webhook 端点，外部事件到达后自动触发任务 |
| **邮箱集成** | 内建 Gmail / Outlook / IMAP 邮箱账号管理与预览 |
| **插件系统** | Python 模块扩展机制，内置 GitHub 插件 |

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.25，Hertz HTTP 框架，GORM，pgx v5 |
| 数据库 | PostgreSQL 16 |
| 缓存 / 队列 | Redis 7 |
| 前端 | Vue 3 + TypeScript，Vite，Arco Design |
| 插件运行时 | Python 3，Octo SDK |
| 容器 | Docker + Docker Compose |

## 快速开始

```bash
git clone <repo-url>
cd OctoManger
docker compose up --build
```

服务就绪后访问 `http://localhost:8080`。

详见 [快速开始](docs/01-quick-start.md)。

## 文档

| 文档 | 说明 |
|------|------|
| [快速开始](docs/01-quick-start.md) | 本地运行与基础操作 |
| [架构说明](docs/02-architecture.md) | 系统设计、模块划分与 API 全览 |
| [配置参考](docs/03-configuration.md) | 全部环境变量说明 |
| [插件开发指南](docs/04-plugin-development.md) | 编写与调试自定义插件 |

## 项目结构

```
OctoManger/
├── apps/
│   ├── api/            # HTTP API 服务入口
│   ├── worker/         # 后台任务调度 & Agent 监管
│   ├── migrate/        # 数据库迁移工具
│   └── web/            # Vue 3 前端应用
├── internal/
│   ├── domains/        # 业务领域（account-types / accounts / jobs /
│   │                   #   agents / email / triggers / plugins / system）
│   └── platform/       # 跨域基础设施（DB、配置、日志、auth、SSE…）
├── db/
│   └── migrations/     # SQL 迁移文件（顺序执行）
├── plugins/
│   ├── modules/        # 已安装的插件目录（热加载）
│   └── sdk/python/     # Python 插件 SDK（octo_runtime）
├── contracts/
│   ├── openapi/        # OpenAPI v2 规范
│   └── plugin-schema/  # 插件 Manifest JSON Schema v2
├── scripts/python/
│   └── modules/github/ # 内置 GitHub 插件
├── configs/            # config.yaml 模板
├── deploy/             # 生产部署辅助文件
└── docker/             # 容器启动脚本
```

## 许可证

MIT
