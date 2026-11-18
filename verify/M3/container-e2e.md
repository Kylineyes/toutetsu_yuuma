# M3 Docker 容器构建与容器内端到端验收（真机执行）

> 执行环境：Debian 12 (bookworm) VM，Docker 28.5.2，Docker Compose v2.40.3
> 部署路径：/home/<user>/project（dev=eef61c8）
> 构建命令：`docker compose build`　启动：`docker compose up -d`

## 1. 构建结果：成功
```
 project-frontend  Built
 project-backend   Built
BUILD_EXIT=0
docker images:
 project-backend   latest  f9e903a8e6f3  555MB
 project-frontend  latest  c75ce7ef911f  48.4MB
```

### 构建期环境适配（本次新增，见 fix/docker-goproxy）
原 `backend/Dockerfile` 直接 `go mod download`，在无法直连 `proxy.golang.org` 的网络下失败：
```
go: github.com/dustin/go-humanize@v1.0.1: Get "https://proxy.golang.org/...": dial tcp: i/o timeout
```
修复：构建阶段增加 `ARG GOPROXY=https://goproxy.cn,direct` + `ENV GOPROXY`，可 `--build-arg` 覆盖。
另：本机 Docker 守护进程原配置了失效镜像站 `docker.fnnas.com`（返回 401），构建前移除了该镜像站（保留代理），基础镜像可正常拉取（此为环境配置，不入仓库）。

## 2. 容器运行状态
```
project-backend-1   project-backend   Up  0.0.0.0:8080->8080/tcp
project-frontend-1  project-frontend  Up  0.0.0.0:8088->80/tcp
backend log: toutetsu_yuuma backend listening on 0.0.0.0:8080 (data=/data)
```

## 3. 容器内端到端冒烟（全部通过）
| 用例 | 结果 |
|---|---|
| GET /api/v1/health（8080 直连） | 200 {"status":"ok"} |
| POST /login 错口令 | 401 |
| POST /login 正确口令 | 200（下发 Cookie） |
| GET /session | {"authenticated":true,"login_required":true} |
| GET /knowledge-bases 空库 | [] （非 null，B7 生效） |
| GET /profiles 空库 | [] |
| GET /tasks 空库 | [] |
| GET /tasks 无 Cookie | 401 |
| 前端 8088 / | 200（nginx/1.27.5，含 <div id="app">） |
| 8088 nginx 反代 /api/v1/health | 200 {"status":"ok"} |

## 4. 真实处理流水线（容器内真实 ffmpeg）
创建任务（上传 3s 测试视频，未选 profile）→ HTTP 201，task id 1795026119818276273-26119。
阶段（真实耗时）：
```
import  done  862ms   (真实 ffmpeg 导入)
audio   done  1117ms  (真实 ffmpeg 抽音)
fuse    done  201ms
summary failed 254ms  未选择配置方案 (profile_id=0)   ← 预期（未选 AI 方案）
```
任务 progress = 30（import10+audio10+fuse10 真实加权聚合，B3 在容器内生效）。

## 5. 文件下载与鉴权
- GET /tasks/{id}/files/fused.srt 带 Cookie → 200（本轮无字幕源，内容为空，符合预期）
- GET /tasks/{id}/files/fused.srt 无 Cookie → 401

## 6. 大文件上传（nginx 反代）
- 容器内 nginx client_max_body_size = 2048m（已确认）
- 20MB 文件经 8088 反代上传 → HTTP 201（无 413）

## 7. 真实浏览器
本机（<本机IP>）Edge headless 访问 http://<部署机IP>:8088/ → 登录页正常渲染（Vue SPA，侧栏 任务/新建任务/配置方案/知识库，登录表单），前端静态资源 200，见 fe_login.png。
nginx access log 记录到 /api/v1/session 200、/api/v1/login 200、/api/v1/tasks 201。

## 结论
M3「Docker 容器构建 + 容器内端到端」在本机判定为 **PASS**：
- 前后端镜像均真实构建成功；
- 两容器健康运行、数据卷落盘 /data；
- 鉴权（Cookie 401/200）、空列表 []、真实 ffmpeg 流水线、progress 聚合、鉴权下载、nginx 反代、大文件上传均通过。
遗留：真实 AI（ASR/vision）仍为降级（无真实 Key），与交接基线一致，非本次 Docker 验收范围。
