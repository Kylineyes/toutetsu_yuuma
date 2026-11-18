# API（/api/v1）

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

## 新建任务（multipart）

- 视频：`video_file` 或 `video_url`
- 字幕（各 1 份，可空）：`subtitle_native / subtitle_ai / subtitle_vision / subtitle_danmu`
- 勾选：`generate_ai_subtitle_via_asr`、`generate_vision_subtitle`
- 方案：`summary_profile_id / asr_profile_id / vision_profile_id / reading_comprehension_profile_id`
- 参数：`screenshot_mode(=simple)、screenshot_interval、reading_depth(T,默认2)、summary_preference`
- 知识库：重复字段 `kb=id:version`（顺序=优先级 1..N）

> 返回上级：[README](../README.md)
