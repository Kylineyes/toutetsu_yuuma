# F2 任务详情 · 证据报告（含 D1 修复）

- 分支：feat/frontend-detail（off feat/frontend-login 7ced3e0）
- 日期：2026-11-18
- 环境：真实浏览器 Playwright 1.65.1 + Edge headless；前端 vite dev http://127.0.0.1:5173（fe_wt 工作树，代理 /api→8080）；后端 dev 源码二进制，PASSWORD=test123。
- 测试任务：真经 API 创建（video_url=http://example.com/x.mp4），pipeline import 阶段真实失败（下载视频 HTTP 404），故有真实 stage 记录；另向 data/task/<id>/output|subs 写入可下载产物以覆盖全部下载按钮。

## D1 修复（合并进本分支）
- 问题：TasksView.vue 原渲染 {{ t.stage }}，但 GET /tasks 返回的 Task 无 stage 字段（已用 tasks_shape.cjs 实证：has stage? false）。
- 修复：移除冗余的“阶段”列（状态列已表达状态），colspan 7→6。
- 验证：任务列表表头实测为 ["任务","视频","状态","进度","创建时间",""]，首行 ["...49957","x.mp4","failed","0%","2026-11-18T05:12:37+08:00","详情删除"]，无 undefined/空白列。见 00-任务列表-D1修复.png。

## 验收项与结果

| 项 | 操作 | 预期 | 实测 | 结果 |
|----|------|------|------|------|
| D1 | 打开任务列表 | 无空白“阶段”列/无 undefined | 表头6列，首行状态=failed、进度=0%，无undefined | ✅ |
| 1 阶段表 | 打开任务详情 | 渲染并行阶段表（阶段/状态/进度/耗时/备注） | import｜failed｜0%｜8.3s｜下载视频 HTTP 404 | ✅ |
| 2/3 实时刷新 | 观察轮询 | 非终态每约3s拉 stages | 终态(failed)时轮询停止；点“重跑”后 6.5s 内 3 次 /stages 请求（约每 3s 一次） | ✅ |
| 4 下载全部 | 带 Cookie 请求 5 个文件 | 全部 200 且有内容 | summary.md 200/115B、knowledge.md 200/55B、fused.srt 200/51B、asr.srt 200/49B、vision.srt 200/51B | ✅ |
| 5 无 Cookie | 匿名下载 | 401 | summary.md → 401 | ✅ |
| 详情渲染 | 详情页 | summary.md/knowledge.md 正文与下载区可见 | 均正确渲染（见 04 截图） | ✅ |

## 关键原始输出
- 阶段表 stageText："/阶段 状态 进度 耗时 备注 | import failed 0% 8.1s 下载视频 HTTP 404"
- 下载(results)：全部 status=200，head 分别为 "# 视频概述"/"## 知识点卡片"/"1"...
- 点击“下载 summary.md”：新开标签 URL=http://127.0.0.1:5173/api/v1/tasks/<id>/files/summary.md；该 URL 带 Cookie 请求 200/115B（浏览器同源自动带 Cookie，HttpOnly 不影响）。

## 截图
- 00-任务列表-D1修复.png
- 01-任务详情页-阶段表.png
- 02-重跑后阶段刷新.png
- 04-下载区与结果(终态).png（含状态/进度条/阶段表/下载按钮/正文）
- 05-点击下载summary后.png

## 说明 / 风险
- 后端 progress 现按阶段聚合；本任务是 import 直接失败，聚合为 0%，符合预期。
- 无真实 AI/ffmpeg，故用真实失败阶段 + 预置产物覆盖下载链路；下载端点本身为真实后端返回。
