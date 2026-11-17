# F4 知识库 · 证据报告

- 分支：feat/frontend-kb（off feat/frontend-login 7ced3e0）
- 环境：真实浏览器 Playwright+Edge；前端 vite dev 5173（fe_wt，作为常驻后台进程）；后端 dev 源码二进制 PASSWORD=test123。
- 变更（KnowledgeBasesView.vue）：新增 fmtSize() 人类可读大小（B/KB/MB）；最新版本行加“最新”标记且删除按钮禁用；错误容器加 .card.err 便于观察。

## 验收项与结果（全部 Pass，真实浏览器）

| 步骤 | 操作 | 期望 | 实测 | 结果 |
|------|------|------|------|------|
| 1 | 输入名称点创建 | 新知识库出现在列表 | 列表出现该 KB，POST /knowledge-bases 生效 | ✅ |
| 2 | 点“版本”查看版本 | 显示版本/文件/术语数/大小/sha/时间 | v3 terms_big.csv 81词 3.4 KB(最新)｜v2 terms2.csv 4词 56 B｜v1 terms1.csv 3词 42 B | ✅ |
| 3 | 删除被任务引用的 v1 | 400 且提示引用 | DELETE /knowledge-bases/1/versions/1 → 400；UI 红字“该版本被 1 个任务引用，不可删除” | ✅ |
| 4 | 删除未被引用的旧版本 v2 | 200 删除成功 | DELETE .../versions/2 → 200；列表不再有 terms2.csv | ✅ |
| 5 | UI 上传新版本到新 KB | 201 并生成 v1 | POST /knowledge-bases/3/versions → 201；版本表出现 v1(最新) ui_terms.csv 3词 59 B | ✅ |

## 关键原始输出（steps.json）
- 步骤2 vTable 见 steps.json；大小列实测 “3.4 KB / 56 B / 42 B”（格式化生效）。
- 步骤3 delReq=[{DELETE .../versions/1, s:400}]，errShown=“该版本被 1 个任务引用，不可删除”。
- 步骤4 delReq=[{DELETE .../versions/2, s:200}]。
- 步骤5 uploadReq=[{POST /knowledge-bases/3/versions, 201}]。

## 证据准备（可复现）
- 用 kb_seed.cjs 预置：创建 KB#1，上传 v1(42B/3词)、v2(56B/4词)、v3(3432B/81词=3.4KB)；并创建 1 个任务引用 KB#1:1，用于制造“被引用版本不可删”的 400。
- 前端上传用真实 CSV 文件 ui_terms.csv。

## 截图
- 01-创建知识库.png
- 02-版本列表含大小.png（3.4 KB/56 B/42 B、最新标记）
- 03_引用版本删除报错.png（红字“该版本被 1 个任务引用，不可删除”）
- 04-删除旧版本后.png
- 05-UI上传新版本.png

## 说明 / 风险
- 最新版本删除按钮已禁用（前端），后端亦拒绝删除最新版本（双重保护）。
- 列表里的 #2/#3 为多次 UI 新建测试数据，属测试残留，非缺陷。
