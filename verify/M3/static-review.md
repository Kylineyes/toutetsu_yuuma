# M3 Docker 静态审查（docker-compose.yml + Dockerfile）

- 审查人：Verification Agent（委托子代理 verify）
- 基线：dev `952fc86`（worktree ver_wt）
- 结论：**Docker 容器构建未验证**——本机无 docker/docker-compose/podman/nerdctl/buildah 二进制，无 docker 服务，无法执行 `docker compose build`。本文件为**静态审查**，非构建证据。

## 一、审查对象
- `docker-compose.yml`（services: backend、frontend）
- `backend/Dockerfile`（多阶段 golang:1.23-bookworm → debian:bookworm-slim）
- `frontend/Dockerfile`（node:20-alpine build → nginx:1.27-alpine）
- `frontend/nginx.conf`（/api 反代、SPA fallback、大请求体）
- `backend/go.mod`/`go.sum`、`.dockerignore`（backend、frontend）、`.env.example`

## 二、正确/合理项（通过静态审查）
1. **后端多阶段构建**：builder `golang:1.23-bookworm` 与 go.mod `go 1.23` 版本一致；先 `COPY go.mod go.sum` 再 `go mod download`（良好缓存分层）；`CGO_ENABLED=0 GOOS=linux` 交叉编译，与纯 Go 驱动 `modernc.org/sqlite` 匹配（无需 cgo/gcc）；产物单文件 `/out/server` 拷入 slim 运行镜像，镜像小。
2. **运行镜像依赖**：`ffmpeg ca-certificates` 已安装（音视频处理 + HTTPS 证书均需要）；`--no-install-recommends` + 清理 apt 列表，合理。
3. **数据卷**：`./data:/data` 挂载，`DATA_DIR=/data`，与 config.go 默认及 main.go `os.MkdirAll` 一致；数据持久化到宿主。
4. **端口**：backend 容器内 8080（compose 映射 8080:8080，与 PORT/EXPOSE 一致）；frontend nginx 80（映射 8088:80），符合 plan.md §2「frontend(80→8088) + backend(8080)」。
5. **反向代理**：nginx `location /api/` → `http://backend:8080`（compose 服务名解析正确）；设置 Host/X-Real-IP；`proxy_read/send_timeout 600s`（适配长时间 AI 任务）；`client_max_body_size 2048m`（大视频上传）。
6. **SPA fallback**：`try_files $uri $uri/ /index.html`，vue-router history 模式刷新不 404，正确。
7. **env 变量映射**：compose environment 的 PASSWORD/API_TOKEN/SESSION_SECRET/MAX_GO_CONCURRENCY/DEFAULT_SCREENSHOT_INTERVAL/MAX_FRAMES/TZ/TMP_DIR 与 backend config.go 读取的键名逐一对应（已核对）。
8. **depends_on**：frontend 依赖 backend（仅启动顺序，非健康就绪）。

## 三、潜在问题 / 改进建议（不阻塞，但建议后续处理）
| # | 级别 | 位置 | 问题 | 建议 |
|---|------|------|------|------|
| A1 | 中 | docker-compose.yml | **无 healthcheck**；`depends_on` 仅启动顺序，backend 未就绪时 frontend 可能先对外，/api 短暂 502 | backend 加 `healthcheck`（curl /api/v1/health），frontend `depends_on: condition: service_healthy` |
| A2 | 中 | frontend/Dockerfile | 用 `npm install` 且只 `COPY package.json`（未 COPY package-lock.json），**依赖版本不确定、可构建性随上游漂移**；还写死 registry npmmirror | 改为 `COPY package.json package-lock.json` + `npm ci`；registry 作为可配置 build-arg |
| A3 | 低 | backend/Dockerfile | 基础镜像 `golang:1.23` / `debian:bookworm-slim` 未固定 digest/tag 小版本，长期可复现性一般 | 可按需固定 digest |
| A4 | 低 | 运行镜像 | 以 root 运行，未创建非 root 用户 | 增 `useradd` + `USER`（安全加固） |
| A5 | 低 | compose | 无 `TMP_DIR` 持久卷（默认 /tmp）；中间产物随容器重建丢失（业务上本就可重建，可接受） | 如需保留可加 tmp 卷 |
| A6 | 提示 | 安全 | API_TOKEN/PASSWORD 通过环境变量注入（compose ${...:-}），空值=开放；生产应显式设置 | 文档提示生产务必设置 PASSWORD/API_TOKEN |
| A7 | 提示 | 资源 | 未设容器 CPU/内存上限与 `MAX_FRAMES` 之外的并发限制 | 按需加 deploy.resources |

## 四、与 plan.md 契约/架构一致性
- 架构 frontend(nginx,80→8088) + backend(8080) + 卷 ./data:/data 与 plan.md §2 一致。
- 端口、卷、反代、SPA、大文件上传、长任务超时均覆盖，设计合理。
- 主要工程化缺口集中在：健康检查（A1）、前端依赖锁（A2）、非 root（A4）。

## 五、环境结论（如实）
- 本机探测：`docker/docker-compose/podman/nerdctl/buildah` 均 NOT FOUND；无 docker 服务；WSL 无发行版。
- **`docker compose build` 与容器内端到端均未验证（环境无 Docker）。** 需在有 Docker 的机器/CI 补跑；届时建议同时验证：镜像可构建、容器内 /health 200、nginx 反代 Cookie 登录、上传大文件、volume 持久化。

## 六、待补跑的验收命令（在有 Docker 的环境）
```
export PASSWORD=changeme API_TOKEN=<token>
docker compose build
docker compose up -d
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8088/api/v1/health   # 期望 200
# 浏览器 http://127.0.0.1:8088 登录→建任务→阶段→下载（同 V2 流程）
docker compose down
```
