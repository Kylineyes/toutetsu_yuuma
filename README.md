# toutetsu_yuuma

一个**长期运行、可 Docker 部署**的「视频阅读」工程：融合多种字幕来源理解视频内容，产出**视频总结**与**知识点**，并支持用领域**知识库**做专有名词纠错。

- 前端：Vue 3 + Vite + vue-router（构建后 nginx 托管）
- 后端：Go（net/http），内置 ffmpeg
- 数据库：SQLite
- AI：按「配置方案」调用（支持 OpenAI Chat/Responses、Anthropic Messages、Gemini、DeepSeek ASR、OpenAI Audio 等协议）

## 功能

- 任务 = 一个视频；异步流水线、**阶段可并行**、断点续跑、分阶段重试，阶段进度与耗时单独记录
- 多字幕来源融合：原生 / AI / 识图 / 弹幕（+ 服务可用 ASR、识图模型自行生成）
- 服务端完成：ffmpeg 抽音轨（16k 单声道 WAV）、VAD 静音切分（≤120s 逐段 ASR）、按间隔截帧
- 产出 `summary.md`（分析时间+概述+按主题分节的具体内容）+ `knowledge.md`（知识点卡片）
- 阅读理解深度 T：总结阶段可多轮「回看画面」（视觉 AI 阅读理解）
- 知识库：结构化术语词典、版本化、内容寻址去重（`blobs/` + `manifest.json`）、可多选、冲突按勾选顺序

## 目录结构
```
toutetsu_yuuma/
├── docker-compose.yml
├── backend/                 Go 服务（内置 ffmpeg）
│   ├── Dockerfile
│   ├── cmd/server/main.go
│   └── internal/{config,db,api,worker,deepseek,media,subtitle,pipeline}/
└── frontend/                Vue3 前端（nginx 托管）
    ├── Dockerfile, nginx.conf
    └── src/{views,api.js,main.js}
```

## 快速开始（Docker）
```bash
export PASSWORD=changeme      # Web UI 登录口令
export API_TOKEN=             # 可选：第三方调用令牌
docker compose up -d --build
# 前端 http://localhost:8088   后端 API http://localhost:8080/api/v1
```

## 本地开发
```bash
cd backend && GOPROXY=https://goproxy.cn,direct go run ./cmd/server
cd frontend && npm install && npm run dev      # http://localhost:5173（代理 /api → :8080）
```

## 环境变量（后端）
| 变量 | 默认 | 说明 |
| --- | --- | --- |
| HOST | 0.0.0.0 | 绑定地址（单地址）|
| PORT | 8080 | 监听端口 |
| DATA_DIR | ./data | 数据根目录 |
| TMP_DIR | 系统临时目录 | 中间产物根（${TMP_DIR}/intermediates/{task}）|
| PASSWORD | 空 | Web UI 登录口令；空=不启用登录 |
| API_TOKEN | 空 | 第三方令牌（X-API-Token / Bearer / ?token=）；空=不校验 |
| SESSION_SECRET | 取 PASSWORD | 登录 Cookie 签名密钥 |
| MAX_GO_CONCURRENCY | 4 | 全局并发上限（ASR 分片 / 识图帧）|
| DEFAULT_SCREENSHOT_INTERVAL | 60 | 默认截图间隔（秒）|
| MAX_FRAMES | 120 | 单任务最多截帧数 |
| TZ | Asia/Shanghai | 时区 |

## 鉴权
- **Web UI**：前端登录页输入 `PASSWORD` → 服务端下发签名 Cookie（HMAC）→ 后续请求凭 Cookie。
- **第三方**：请求头 `X-API-Token: <API_TOKEN>`（或 `Authorization: Bearer`；下载可 `?token=`）。
- 免鉴权：`/health`、`/login`、`/session`。两条通道都空 = 完全开放。

## 配置方案（Profile）
每类能力一份：`capability` = summary/asr/vision/reading；建任务时每类各选一个。
- 字段：`protocol / base_url / api_key / model / temperature / timeout_seconds / extra(JSON)`
- chat 协议（summary/vision/reading）：`openai_chat / openai_responses / anthropic_messages / gemini`
- audio 协议（asr）：`deepseek_asr`(/v1/tools/asr) / `openai_audio`(/v1/audio/transcriptions)

## API（/api/v1）
| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | /health, /meta | 健康 / 元信息 |
| POST | /login, /logout ; GET /session | 登录 / 登出 / 会话状态 |
| GET/POST | /profiles?capability= | 方案列表/新建 |
| GET/PUT/DELETE | /profiles/{id} | 方案详情/改/删 |
| GET/POST | /knowledge-bases | 知识库列表/新建 |
| GET/DELETE | /knowledge-bases/{id} | 详情/删除 |
| GET/POST | /knowledge-bases/{id}/versions | 版本列表 / 上传新版本(multipart file) |
| GET/DELETE | /knowledge-bases/{id}/versions/{v} | 版本信息 / 删除旧版本(未引用且非最新) |
| POST | /tasks | 新建任务(multipart) |
| GET | /tasks ; /tasks/{id} ; /tasks/{id}/stages | 列表 / 详情 / 阶段 |
| GET | /tasks/{id}/result | 结构化结果(含 stages/cost/md) |
| GET | /tasks/{id}/files/{name} | 下载 summary.md/knowledge.md/fused.srt/asr.srt/vision.srt |
| POST | /tasks/{id}/retry | 重跑 |
| DELETE | /tasks/{id}/cache | 清理 tmp 中间文件(仅 paused/failed/done) |
| DELETE | /tasks/{id} | 删除任务 |

### 新建任务（multipart）
- 视频：`video_file` 或 `video_url`
- 字幕（各 1 份，可空）：`subtitle_native / subtitle_ai / subtitle_vision / subtitle_danmu`
- 勾选：`generate_ai_subtitle_via_asr`、`generate_vision_subtitle`
- 方案：`summary_profile_id / asr_profile_id / vision_profile_id / reading_comprehension_profile_id`
- 参数：`screenshot_mode(=simple)、screenshot_interval、reading_depth(T,默认2)、summary_preference`
- 知识库：重复字段 `kb=id:version`（顺序=优先级 1..N）

## 数据布局
```
/data/db/app.db
/data/knowledge-bases/blobs/{sha前2}/{sha}      # 内容寻址去重
/data/knowledge-bases/{kbId}/v{ver}/manifest.json
/tmp/intermediates/{taskId}/                     # 音频/分片/截图帧（TMP_DIR 下）
/data/task/{taskId}/upload|subs|output/
```

## v1 限制
- 只处理视频源文件 / 直链，不做平台页解析（yt-dlp 后续）
- DeepSeek ASR 无词级时间戳 → ASR 字幕为分片级近似；fused.srt 以原字幕时间轴为准
- 截图仅 simple（固定间隔）；paused 状态预留未实现
- 成本仅记录展示不限制；单租户
