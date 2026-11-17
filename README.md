# toutetsu_yuuma

一个**长期运行、可 Docker 部署**的「视频阅读」工程：融合多种字幕来源理解视频内容，产出**视频总结**与**知识点**，并支持用领域**知识库**做专有名词纠错。

- 前端：Vue 3 + Vite + vue-router（构建后由 nginx 托管）
- 后端：Go（net/http），内置 ffmpeg
- 数据库：SQLite
- AI：DeepSeek（ASR 语音识别 / 识图 / 文本总结，均为可配置的方案）

## 功能

- 任务 = 一个视频（最细分析粒度），异步流水线、断点续跑、分阶段重试
- 多字幕来源融合：原生字幕 / AI 字幕 / 识图字幕 / 弹幕（+ 服务可用 ASR/识图自行生成 AI/识图字幕）
- 音视频处理全在服务端：ffmpeg 抽音轨（16k 单声道 WAV）、VAD 静音切分（≤120s 逐段 ASR）、按间隔截帧
- 生成：`summary.md`（分析时间 + 概述 + 按主题分节的具体内容）+ `knowledge.md`（知识点卡片条目化）
- 阅读理解深度 T：总结阶段可让总结 AI 多轮「回看画面」（视觉 AI 阅读理解）
- 知识库：结构化术语词典，版本化（每次提交刷新版本），任务可多选、按勾选顺序决定冲突优先级；正文存本地目录，DB 只做版本索引
- 可下载产物：summary.md / knowledge.md / 统一字幕 fused.srt / ASR字幕 / 识图字幕

## 目录结构

```
toutetsu_yuuma/
├── docker-compose.yml
├── backend/                 Go 服务（内置 ffmpeg）
│   ├── Dockerfile
│   ├── cmd/server/main.go
│   └── internal/
│       ├── config/ db/ api/ worker/
│       ├── deepseek/         ASR / Vision / Chat 客户端
│       ├── media/            ffmpeg 抽音轨、切分、截帧
│       ├── subtitle/         SRT/VTT/ASS/TXT 解析 + 融合 + 术语纠错
│       └── pipeline/         任务流水线（T 轮阅读、弹幕清洗、知识库纠错）
└── frontend/                Vue3 前端（构建后 nginx 托管）
    ├── Dockerfile, nginx.conf
    └── src/ (views: 任务/新建/详情/方案/知识库)
```

## 快速开始（Docker）

```bash
export API_TOKEN=changeme        # 可选
docker compose up -d --build
# 前端: http://localhost:8088   后端 API: http://localhost:8080/api/v1
```

数据持久化在 `./data:/data`。首次启动自动建表。

## 本地开发

```bash
cd backend
GOPROXY=https://goproxy.cn,direct go run ./cmd/server   # 默认 :8080, DATA_DIR=./data
cd frontend
npm install && npm run dev        # http://localhost:5173 （已代理 /api → :8080）
```

## 环境变量（后端）

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| PORT | 8080 | 监听端口 |
| DATA_DIR | ./data | 数据根目录（db / task / knowledge-bases） |
| TMP_DIR | 系统临时目录 | 中间产物根目录（音频/分片/截图帧，落 TMP_DIR/intermediates） |
| DEFAULT_SCREENSHOT_INTERVAL | 60 | 默认截图间隔（秒）|
| MAX_FRAMES | 120 | 单任务最多截帧数 |
| TZ | Asia/Shanghai | 时区 |

## 鉴权（API Token）

API Token **不使用环境变量**，而是**明文存储于数据库 `settings` 表**（`key=api_token`）。可通过前端「设置」页或 `PUT /api/v1/settings` 设置；`GET /api/v1/settings` 读取。设置后，除 `/health`、`/settings` 外的所有 `/api` 请求需带 `X-API-Token`（或 `Authorization: Bearer`，下载文件也支持 `?token=`）。留空则不启用鉴权。

## 配置方案（Profile）

四类能力各自一份连接配置，建任务时每类各选一个，可各用不同 token：

| 能力 | 用途 |
| --- | --- |
| summary | 生成总结 + 知识点（文本模型）|
| asr | 音频转写（POST /v1/tools/asr，WAV/Opus，≤120s/≤8MiB 逐段）|
| vision | 识图字幕生成（deepseek-flash 视觉）|
| reading | 阅读理解（总结阶段回看画面，视觉）|

> DeepSeek 兼容 OpenAI 的 /chat/completions；ASR 走 /v1/tools/asr。Base URL 默认 https://api.deepseek.com。

## API（/api/v1）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | /health | 健康检查 |
| GET | /meta | 支持格式、默认参数 |
| GET | /settings | 读取设置（含明文 api_token）|
| PUT | /settings | 保存设置（api_token，明文存 DB）|
| GET/POST | /profiles?capability= | 方案列表/新建 |
| GET/PUT/DELETE | /profiles/{id} | 方案详情/更新/删除 |
| GET/POST | /knowledge-bases | 知识库列表/新建 |
| GET/DELETE | /knowledge-bases/{id} | 知识库详情/删除 |
| GET/POST | /knowledge-bases/{id}/versions | 版本列表 / 上传新版本（multipart file）|
| GET | /knowledge-bases/{id}/versions/{v} | 版本元信息 |
| POST | /tasks | 新建任务（multipart，见下）|
| GET | /tasks | 任务列表 |
| GET | /tasks/{id} | 任务状态/阶段/进度 |
| GET | /tasks/{id}/result | 结构化结果（summary_md/knowledge_md/cost…）|
| GET | /tasks/{id}/files/{name} | 下载 summary.md / knowledge.md / fused.srt / asr.srt / vision.srt |
| POST | /tasks/{id}/retry | 重跑 |
| DELETE | /tasks/{id}/cache | 清理该任务 tmp 中间文件（仅 paused/failed/done）|
| DELETE | /tasks/{id} | 删除 |

### 新建任务（multipart 字段）

- 视频：video_file（上传）或 video_url（直链，二选一）
- 字幕（各 1 份，可空）：subtitle_native / subtitle_ai / subtitle_vision / subtitle_danmu
- 勾选：generate_ai_subtitle_via_asr、generate_vision_subtitle（"1"/"true"）
- 方案：summary_profile_id / asr_profile_id / vision_profile_id / reading_comprehension_profile_id
- 参数：screenshot_mode(=simple)、screenshot_interval(秒)、reading_depth(T, 默认2)、summary_preference(可空)
- 知识库：重复字段 kb，值 id:version（出现顺序 = 优先级）

## 数据布局

```
/data/db/app.db
/data/knowledge-bases/{kbId}/v{version}/     知识库术语文件（持久）
/tmp/intermediates/{taskId}/                 音频/分片/截图帧（临时）
/tmp/intermediates/{taskId}/turn/{round}/    每轮回看中间产物
/data/task/{taskId}/upload/                  视频 + 上传字幕
/data/task/{taskId}/subs/                    native/asr/vision 字幕 + fused.srt
/data/task/{taskId}/output/                  summary.md、knowledge.md
```

## v1 范围与限制

- 只处理视频源文件 / 直链，不做平台页解析（yt-dlp 留后续）
- DeepSeek ASR 无词级时间戳 → ASR 字幕时间轴为分片级近似；fused.srt 以原生/AI 字幕精确时间轴为准，ASR 只补文本
- 截图仅 simple（固定间隔）模式；complex 预留
- 成本仅记录与展示，不自动中止
- 单租户、可选 API Token；无多用户账号体系
