# V4 前后端契约一致性静态审计

- 审计人：Verification Agent（委托子代理 verify）
- 分支：verify/v3-v4-audit（基于 dev a98f87e）
- 审计方式：**只读静态比对** frontend/src/** 与 backend/internal/api/**、db 结构体、config，逐条对照 plan.md 第 6 节接口契约。
- 说明：本报告为静态审计；运行时行为待 lead 通知后由 V1(curl) / V2(WebUI) 实测确认。

## 一、契约逐条比对（plan.md §6）

| # | 契约条目 | 后端实现 | 前端调用 | 结论 |
|---|----------|----------|----------|------|
| 1 | 鉴权：Cookie(登录) 或 `X-API-Token`；免鉴权 /health /login /session | api.go withAuth：接受 `X-API-Token`、`Authorization: Bearer`、`?token=`；session cookie(HMAC)；exempt: health/login/session | api.js：登录后浏览器自动带 cookie；未发 token（Web 端合理） | **一致** |
| 2 | 创建任务 multipart 字段 video_file/video_url、subtitle_native/ai/vision/danmu、generate_*、四个 profile_id、screenshot_mode/interval、reading_depth、summary_preference、kb=id:version | api_tasks.go createTask 全部按此读取 | TaskCreate.vue submit() 全部按此 append | **一致** |
| 3 | 任务结果 GET /tasks/{id}/result → {task,cost,subtitles,knowledge_bases,stages,summary_md,knowledge_md} | taskResult 精确返回这 7 个键 | api.taskResult；TaskDetail 用 result.cost/summary_md/knowledge_md | **一致** |
| 4 | 阶段 GET /tasks/{id}/stages → [{stage,status,progress,duration_ms,message,error}] | taskStages 返回 TaskStage（含上述字段，另多 id/task_id/started_at/finished_at） | TaskDetail 用 s.stage/status/progress/duration_ms/message/error | **一致**（多字段无害） |
| 5 | 文件下载 GET /tasks/{id}/files/{summary.md|knowledge.md|fused.srt|asr.srt|vision.srt} | taskFile 白名单 5 个文件名，其余 404 | api.fileUrl 拼 /tasks/{id}/files/{name}；TaskDetail 下载按钮 5 个同名 | **一致** |
| 6 | screenshot_mode 取值（契约文字写 single） | 后端默认 "simple" | 前端下拉仅 "simple" | **偏差 D2（文档 vs 实现，详见下）** |

## 二、发现的偏差表

| 编号 | 级别 | 位置 | 现象 | 影响 | 建议 |
|------|------|------|------|------|------|
| **D1** | 中 | frontend/src/views/TasksView.vue:28 | 列表页渲染 `{{ t.stage }}`，但后端 `GET /tasks` 返回 `[]Task`（db.Task 结构体**无 stage 字段**），stage 只存在于 `/tasks/{id}/stages` | 任务列表“阶段”列**永远为空**（undefined）。属前端字段期望与后端返回不符 | 前端删除该列，或后端 ListTasks 聚合当前阶段返回；任选其一并更新契约 |
| **D2** | 低 | plan.md §6 vs backend/api_tasks.go:34 vs TaskCreate.vue:19,111 | 契约文字 `screenshot_mode=single`，但后端默认值与前端下拉框均为 `simple` | 若第三方按文档传 `single`，后端会原样存 `single`（不校验），但前端 UI 只产生 `simple`；当前无功能报错，仅**文档与实现不一致** | 统一为 `simple` 并改 plan.md，或统一为 `single` |
| **D3** | 低 | frontend/src/api.js:16 | `session()` 直接用 `.json()`，未走 handle()，网络/解析异常不抛给 UI（router 里 catch 忽略） | 后端异常时静默放行到受保护页，再由具体接口 401 兜底；非致命 | 可接受；建议记入已知行为 |
| **D4** | 低 | backend/api_knowledge.go deleteKB | `DELETE /knowledge-bases/{id}` 无引用校验，直接删 KB 行；仅版本级删除有“被任务引用”保护 | 删除被任务引用的 KB 后，task_knowledge_bases 残留孤儿行（应用层级联只在 DeleteTask 时清理），且无外键 | 建议 deleteKB 增加引用检查或文档声明；非本轮阻塞 |
| **D5** | 提示 | backend/api.go:208-221 vs meta | `asrProtocols` 含 `openai_whisper`，但 `/meta` 的 asr_protocols 仅列 deepseek_asr/openai_audio；前端 asrProtocols 也仅 2 项 | 仅内部多识别一个协议，UI 不展示，无影响 | 无需处理，记录 |

## 三、字段级核对（关键写路径，均一致）

- **createProfile**：前端 payload {capability,protocol,name,base_url,model,api_key,temperature(可为 null),timeout_seconds,extra} → 后端 db.Profile JSON tag 完全匹配（temperature 为 *float64 接受 null）。✅
- **profiles 列表**：后端对 api_key 统一脱敏为 "***"；前端不展示 api_key。✅
- **createKB**：{name} ✅；**uploadKB**：multipart 字段 `file` ✅；**kbVersions**：`[{id,kb_id,version,dir_path,file_name,sha256,term_count,size_bytes,created_at}]`，前端取 version/term_count/size_bytes/sha256/file_name ✅；版本倒序 DESC，前端取 vs[0] 为最新 ✅。
- **createTask**：前端 `reading_comprehension_profile_id` 上传 → 后端存入 ReadingProfileID（json reading_profile_id）。输入键名一致 ✅。
- **kb=id:version**：前端 `kid + ':' + ver`，后端 SplitN(":",2) 解析，priority 按顺序 i+1 ✅。
- **retry/cleanupCache/deleteTask**：POST /retry、DELETE /cache（仅 done/failed/paused）、DELETE /tasks/{id}，前端 api.js 路径与方法均一致 ✅。
- **cleanupCache 前端提示文案**“删除该任务的临时中间文件（tmp 缓存）”，与后端删除 `TMP_DIR/intermediates/{id}` 语义一致 ✅。

## 四、路由/前缀核对
- 前端 base='/api/v1'，所有调用均带该前缀；后端全部注册于 /api/v1/* ✅。
- dev 环境 vite proxy '/api'→localhost:8080 ✅；生产 nginx 需反代 /api（见 frontend/nginx.conf，V3/容器阶段核对）。

## 五、结论
- 主契约（鉴权、创建、结果、阶段、下载）**结构与键名一致**，无阻断性字段错配。
- 需 dev 修正：**D1（任务列表 stage 列恒空）** 为唯一可见功能性偏差，建议 frontend 处理。
- 文档/实现不一致：**D2（single vs simple）** 建议统一并回写 plan.md。
- 其余 D3/D4/D5 为低风险/提示，记录备查，不阻塞。
