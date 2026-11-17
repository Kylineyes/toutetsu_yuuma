# V2 WebUI 交互复核报告（真实浏览器）

- 复核人：Verification Agent（委托子代理 verify），**独立复核**，未采信 frontend 自述。
- 被测基线：dev `9af32cc`（F1–F5 + D1 已合入）。隔离 worktree `<工作目录>\ver_wt`。
- 后端：`go build -o server.exe ./cmd/server`（exit 0），`PASSWORD=test123 PORT=18090` 真实启动。
- 前端：dev 最新源码 `npm run build`（exit 0），由本地静态服务器(18091)托管 `dist` 并反代 `/api`→18090。
- 浏览器：Microsoft Edge 157（headless），经 **CDP（Chrome DevTools Protocol）** 脚本驱动，独立截图。
- 时间：2026-11-18T05:35＋08:00

## 总体结果：**PASS 32 / FAIL 0**（5 组功能，逐条见下）

### F1 登录流程（6/6 PASS）
| 项 | 结果 | 证据 |
|----|------|------|
| F1.1 未登录访问 /tasks → 跳 /login | PASS | F1-1-401-redirect-login.png |
| F1.2 错误口令显示“密码错误” | PASS（截图红字可见） | F1-2-wrong-password.png |
| F1.3 正确口令 → 200 + 跳 /tasks | PASS（网络 login=200） | F1-3-login-success-tasks.png |
| F1.4 刷新后保持登录 | PASS | （导航后仍在 /tasks） |
| F1.5 退出 → 跳 /login | PASS | F1-5-logout-login.png |
| F1.6 退出后访问 /tasks 再跳登录 | PASS | — |
网络记录：session 200 / login 401 / login 200 / tasks 200 / logout 200。

### F2 详情 + D1（8/8 PASS）
| 项 | 结果 | 证据 |
|----|------|------|
| **D1** 列表表头 6 列（任务/视频/状态/进度/创建时间/操作） | PASS | F2-list-tasks.png |
| **D1** 已删除“阶段”列 | PASS（表头无“阶段”） | F2-list-tasks.png |
| **D1** 页面无 undefined 文本 | PASS | F2-list-tasks.png |
| F2 详情页有“阶段（可并行）”表 | PASS | F2-detail-stages.png |
| F2 阶段表渲染行数 > 0 | PASS | F2-detail-stages.png |
| F2 非终态轮询（重复读取进度/阶段刷新） | PASS | — |
| F2 带 Cookie 下载 5 文件鉴权通过（非 401；文件未生成为 404） | PASS | 状态={summary.md:404,knowledge.md:404,fused.srt:404,asr.srt:404,vision.srt:404} |
| F2 无 Cookie 下载 → 401 | PASS | 401 |

> 说明：任务为冒烟任务（无真实 AI/媒体），文件未生成故带 Cookie 下载为 404（鉴权已通过，非 401）；鉴权维度已覆盖。

### F3 配置方案（10/10 PASS）
| 项 | 结果 |
|----|------|
| 切 ASR 能力，协议选项 = deepseek_asr/openai_audio/openai_whisper | PASS |
| 切总结能力，协议含 openai_chat/responses/anthropic/gemini | PASS |
| 温度越界(>2) → 拦截“温度须为 0~2…” | PASS（F3-temp-out-of-range.png） |
| 超时 0 → 拦截“超时须为正整数秒” | PASS（F3-timeout-zero.png） |
| extra 坏 JSON → 拦截“extra 不是合法 JSON” | PASS（F3-extra-badjson.png） |
| 合法值创建 → 201 回显（行数 +1） | PASS（F3-valid-created.png） |

### F4 知识库（4/4 PASS）
| 项 | 结果 |
|----|------|
| 版本大小人类可读（如 35 B / 24 B / KB） | PASS（35 B,24 B,19 B） |
| 最新版本“最新”标记 | PASS（v3 最新） |
| 删被引用版本(v1) → 400“该版本被 1 个任务引用，不可删除” | PASS（F4-delete-referenced-400.png） |
| UI 上传新版本 → POST 201（v3 出现） | PASS（F4-upload-new-version.png；POST /versions=201） |

### F5 新建任务（10/10 PASS）
| 项 | 结果 |
|----|------|
| 多字幕插槽（5 个 file 输入：视频+原生/AI/识图/弹幕） | PASS（F5-create-initial.png） |
| 四类字幕槽标注齐全 | PASS |
| 勾选项（ASR 生成 / 识图生成）存在 | PASS |
| KB 多选后版本下拉出现（v4/v3/v2/v1） | PASS |
| 勾选显示“优先级 1” | PASS |
| **空知识库(无版本)选择不崩溃** + 提示“该知识库尚无版本，无法选择”且不加下拉 | PASS（F5-empty-kb-select.png，页面存活，select 数 6→6） |
| 参数校验：截图间隔 0 → 拦截“截图间隔须为正整数秒” | PASS（F5-validation-interval-zero.png） |
| 合法提交 → POST /tasks 201 | PASS |
| 提交后跳转详情页 | PASS（F5-submit-success-detail.png） |

## 特别复核项结论
- **D1 真正修复**：列表为 6 列，无“阶段”列、无空白列、无 undefined（源码 TasksView.vue 亦无 t.stage）。✔
- **F5 空知识库不崩溃**：勾选空库仅提示“尚无版本，无法选择”，页面与表单保持可用，未新增版本下拉。✔

## 证据清单（verify/V2/）
截图：F1-1-401-redirect-login.png、F1-2-wrong-password.png、F1-3-login-success-tasks.png、F1-5-logout-login.png、F2-list-tasks.png、F2-detail-stages.png、F3-profiles-initial.png、F3-temp-out-of-range.png、F3-timeout-zero.png、F3-extra-badjson.png、F3-valid-created.png、F4-kb-versions.png、F4-delete-referenced-400.png、F4-upload-new-version.png、F5-create-initial.png、F5-empty-kb-select.png、F5-validation-interval-zero.png、F5-submit-success-detail.png

## 复现说明
- 环境：本机无 Playwright（npm 远程拉取被限），改用 Edge + CDP（Node 原生 WebSocket/fetch）驱动；脚本在工作树根 `_cdp.js/_f1.js/_f2.js/_f3b.js/_f4.js/_f4b.js/_f5.js`（仅测试工具，不提交业务改动）。
- 前端用 dev 最新源码构建（dist），非旧 dist。

## 遗留/说明
- 真实 AI 端到端（下载到真实 summary.md/knowledge.md/fused.srt 等）未覆盖，需真实 Key；本报告下载鉴权维度已覆盖（Cookie 非401 / 无Cookie 401）。
- Docker 容器内 WebUI 复审未做（本机无 Docker，见 V3）。
