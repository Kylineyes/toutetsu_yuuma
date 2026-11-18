# toutetsu_yuuma

> 项目名 **Toutetsu Yuuma** 取自「**饕餮尤魔**」的日语读音（とうてつ ゆうま）。

一个**长期运行、可 Docker 部署**的「视频阅读」工程：融合多种字幕来源理解视频内容，产出**视频总结**与**知识点**，并支持用领域**知识库**做专有名词纠错。

> 前端 Vue 3 + Vite（nginx 托管）｜后端 Go（net/http，内置 ffmpeg）｜SQLite｜AI 走「配置方案」（多协议）

## 文档目录

按主题拆分，详见各分文档：

| 主题 | 文档 | 内容 |
| --- | --- | --- |
| 项目概览 | [docs/overview.md](docs/overview.md) | 简介、技术栈、功能特性 |
| 架构与数据布局 | [docs/architecture.md](docs/architecture.md) | 目录结构、数据落盘布局 |
| 快速开始与本地开发 | [docs/quickstart.md](docs/quickstart.md) | Docker 启动、本地开发 |
| 配置与鉴权 | [docs/configuration.md](docs/configuration.md) | 环境变量、鉴权方式、配置方案 Profile |
| API | [docs/api.md](docs/api.md) | /api/v1 接口表、新建任务 multipart 字段 |
| v1 限制 | [docs/limitations.md](docs/limitations.md) | 当前版本能力边界 |

## 其它文档

- [视频处理流程图](docs/视频处理流程图.md)：任务从「被接受」到各处理阶段的完整流程
- [交接报告](docs/交接报告.md)、[换机交接说明](docs/换机交接说明.md)：项目验收与接手说明
- [AGENTS.md](docs/AGENTS.md)：协作与提交规范

## 一分钟上手

```bash
export PASSWORD=changeme      # Web UI 登录口令
export API_TOKEN=             # 可选：第三方调用令牌
docker compose up -d --build
# 前端 http://localhost:8088   后端 API http://localhost:8080/api/v1
```

详见 [快速开始](docs/quickstart.md)。
