# 前端 × 真实后端 × Magpie 网关联调（F-I1）

- 目的：验证重构后的前端在**真实后端 + 真实模型**下可用，且 AI 配置方案能打通
- 环境：Windows；Go 1.27（本地编译 server.exe）、Node 26、ffmpeg 6/7；浏览器 Edge（Playwright）

## 拓扑

| 角色 | 地址 | 说明 |
|---|---|---|
| 后端 | http://127.0.0.1:8080 | 本地编译产物；DATA_DIR/TMP_DIR 指向系统临时目录；PASSWORD 与 API_TOKEN 仅以环境变量注入，不落仓库 |
| 前端 | http://127.0.0.1:5173 | vite dev，/api 代理到 8080 |
| 模型网关 | Magpie（内网 :30131，路径 /v1） | 配置方案 base_url 写到 /v1，客户端再拼 /chat/completions |

## 素材

- 60 秒 640×360 合成视频（纯色画面 + 440Hz 音轨，ffmpeg 生成）
- 12 条中文原生字幕（夏布利霞多丽品鉴，覆盖产区/香气/口感/配餐/温度/陈年）

## 流程与结果

| 步骤 | 命令/动作 | 结果 |
|---|---|---|
| 建配置方案 | POST /api/v1/profiles（summary，openai_chat，base_url=网关/v1，temperature 0.2） | 201，id=1 |
| 建任务 | POST /api/v1/tasks（video_file + subtitle_native，summary_profile_id=1，reading_depth=1） | 201，任务进入队列 |
| 阶段执行 | GET /tasks/{id}/stages 轮询 | import/audio/fuse/summary = done（asr/frame/vision 因已有原生字幕未触发） |
| 总结 | summary 阶段调用网关 | 26.8s 完成，prompt 516 / completion 6500 tokens |
| 产物 | GET /tasks/{id}/result | summary.md 833 字符、knowledge.md 1927 字符，内容与字幕主题一致 |

## 前端实跑（Playwright，真实数据，非 mock）

| 检查 | 结果 |
|---|---|
| 未登录访问 /tasks | 跳转 /login |
| 口令登录 | 进入 /tasks，任务表 1 行（真实任务 ID） |
| 任务详情 | 2 个产物块渲染，阶段表 4 行，token 成本 516/6500 展示正确 |
| 主题切换 | light → dark，刷新后仍 dark（ty-theme 持久化） |
| 会话保持 | 刷新后仍在详情页，未被踢回登录 |
| 配置方案页 | 1 行真实 profile，名称/模型/base_url 正确回显，API Key 显示为 *** |
| 知识库页（真实空库） | 正确渲染空态「暂无知识库」，无 null 崩溃 |
| 新建任务页 | 方案下拉 2 个选项（不指定 + 唯一方案） |
| 控制台 | 修复 favicon 404 后 **0 error / 0 失败请求** |

## 本轮修掉的问题

| 问题 | 表现 | 修复 |
|---|---|---|
| 缺 favicon | 每次首屏浏览器请求 /favicon.ico 404，控制台报错 | index.html 内联 SVG data-URI 图标（TY 蓝底），无新增文件 |

## 截图

| 文件 | 内容 |
|---|---|
| 01-login.png | 未登录跳转后的登录页 |
| 02-tasks-real.png | 真实任务列表（1 行） |
| 03-task-detail-real.png | 真实详情：阶段表 + 成本 + summary + knowledge |
| 04-task-detail-dark.png | 同上，暗色主题 |
| 05-profiles-real.png | 真实配置方案 |
| 06-knowledge-empty.png | 真实空库空态 |
| 07-task-create-real.png | 新建任务（真实方案下拉） |
| report.json | 上述断言的机器可读结果 |

## 未覆盖

- ASR 通道：未配置 ASR 方案（本次用原生字幕跳过 asr 阶段）。
- 识图/OCR：未配置 vision 方案。
- 知识库纠错：空库，未验证术语注入。
- 网关能力边界：只验证 chat/completions；未验证该网关是否提供 /v1/tools/asr。
- 并发与限流：单任务串行验证。
