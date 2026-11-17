# V3 构建复跑报告

- 报告人：Verification Agent（委托子代理 verify）；本报告所有结果均为**独立复跑**，未采信 dev 自述。
- 基线：dev 最新 commit **a98f87e**（隔离 git worktree `<工作目录>\ver_wt`，分支 verify/v3-v4-audit，HEAD=a98f87e）。
- 环境：Windows / Go 1.27.4 / Node v26.13.1 / npm 12.0.2 / **本机无 Docker**。
- 时间：2026-11-18T05:11:38+08:00

## 结果总览

| 项目 | 命令 | 结果 | 证据 |
|------|------|------|------|
| 后端编译 | `go build ./...` | **PASS** (exit 0) | `verify/V3/build-a98f87e.log` |
| 后端静态检查 | `go vet ./...` | **PASS** (exit 0) | `verify/V3/build-a98f87e.log` |
| 前端构建 | `npm run build` | **PASS** (exit 0，32 modules，dist 产物) | `verify/V3/build-a98f87e.log` |
| Docker 镜像构建 | `docker compose build` | **BLOCKED / 未验证** — 本机未安装 Docker | `verify/V3/docker-build.log` |

## 证据要点（原始输出）
`go build ./...`：无输出，`BUILD_EXIT=0`。
`go vet ./...`：无输出，`VET_EXIT=0`。
`npm run build`（vite 5.4.21）：
```
✓ 32 modules transformed.
dist/index.html                   0.42 kB
dist/assets/index-C-wctoaI.css    1.92 kB
dist/assets/index-B7RMSnCn.js   120.09 kB │ gzip: 45.33 kB
✓ built in 770ms
NPMBUILD_EXIT=0
```

> 关于 npm clean install：在隔离 worktree 中直接 `npm ci` 被 npm 安全策略拦截（`EALLOWREMOTE: Fetching packages of type "remote" have been disabled`，registry 的 tgz 远程拉取受限）。故复用主工作区已成功安装的 `frontend/node_modules` 进行构建。此为环境安装限制，不影响“源码可在 a98f87e 下成功构建”的结论。

## docker compose build（未验证，如实报告）
探测证据（`verify/V3/docker-build.log`）：
- `docker` / `docker-compose` / `podman` 命令均不存在；
- `C:\Program Files\Docker`、`C:\ProgramData\DockerDesktop` 均不存在；
- 无 `*docker*` 服务；
- WSL 无已安装 Linux 发行版。

**结论：本机不具备 Docker 环境，`docker compose build` 无法执行，此项标记为 BLOCKED/未验证，不得视为通过；需在有 Docker 的机器/CI 补跑。**

静态核对（仅供参考，非构建证据）：
- `backend/Dockerfile`：多阶段 `golang:1.23-bookworm`；go.mod `go 1.23` 与之一致；`go.sum` 存在；CGO_ENABLED=0 交叉编译 linux，与纯 Go 驱动 modernc.org/sqlite 匹配；运行镜像 debian:bookworm-slim + ffmpeg/ca-certificates。预期可构建。
- `frontend/Dockerfile`：node:20-alpine 构建 + nginx:1.27-alpine；使用 `npm install`（非 npm ci），未 COPY package-lock（轻微最佳实践偏差，不影响构建）。
- `docker-compose.yml`：backend(8080)、frontend(8088:80)、卷 `./data:/data`，env 与 config.go 变量名一致。

## 环境/协作备注
- 主仓库目录 `<工作目录>\toutetsu_yuuma` 由 Backend 独占并在多分支间切换；为避免证据被并发改动污染，本 V3 全部构建在隔离 worktree `ver_wt`（基于 verify/v3-v4-audit @ a98f87e）内完成，HEAD 与源码经 `git rev-parse HEAD` 核实为 a98f87e。
