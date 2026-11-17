# F5 新建任务 · 证据报告

- 分支：feat/frontend-create（off feat/frontend-login 7ced3e0）
- 环境：真实浏览器 Playwright+Edge；前端 vite dev 5173（fe_wt）；后端 dev 源码二进制 PASSWORD=test123。

## 发现并修复的真实缺陷
- 缺陷：勾选第二个“空知识库（最新 v0）”时，api.kbVersions 返回 null（后端 ListKBVersions 对空库返回 null），toggleKB 内 vs.length 抛 “Cannot read properties of null (reading 'length')”，导致该次选择整体失败（Vue 原生事件处理器异常）。
- 修复（TaskCreate.vue）：kbVersions 结果做 null/非数组兜底；对无版本（latest_version<=0）的库直接拒绝选择并提示“该知识库尚无版本，无法选择”；取消选择时同时清理 versionsMap。修复后多库选择正常。

## 变更（TaskCreate.vue）
1) 参数校验：截图间隔须为正整数秒、阅读深度须为非负整数（提交前拦截）；
2) 知识库选中显示“优先级 N”徽标（勾选顺序即优先级）；
3) 修复空库 null 崩溃（见上）；
4) 错误容器加 .card.err。

## 验收项与结果（全部 Pass，真实浏览器）

| 步骤 | 操作 | 期望 | 实测 | 结果 |
|------|------|------|------|------|
| 1 | 打开新建页 | 视频+4字幕插槽、2生成勾选 | 5 个 file 输入（视频+原生/AI/识图/弹幕）、5 个复选框 | ✅ |
| 2 | 截图间隔填 0 提交 | 拒绝并提示 | “截图间隔须为正整数秒” | ✅ |
| 3 | 阅读深度填 -1 提交 | 拒绝并提示 | “阅读深度须为非负整数” | ✅ |
| 4 | KB 勾选 KB#1 与 KB#3 | 多选生效、显示优先级、各自版本可选 | 优先级 1 / 优先级 2；2 个版本下拉 | ✅ |
| 5 | 勾选两个生成开关并提交 | 201 跳详情，kb=id:version 正确、开关为 1 | POST /tasks→201；跳 /tasks/<id>；kbParams=["1:3","3:1"]；genAI=1 genVis=1 | ✅ |

## 关键原始输出（steps.json）
- 步骤4 priorityTags=["优先级 1","优先级 2"]，versionSelects=2。
- 步骤5 submitStatus=201，kbParams=["1:3","3:1"]，genAI=true，genVis=true，finalUrl=/tasks/1794950919175447000-50919。

## 证据准备
- 复用 F4 预置的 KB#1(v1,v3) 与 KB#3(v1)；KB#2 为空库（用于验证空库不可选/不崩溃）。
- 提交后跳转到真实任务详情，可见 import 阶段 running（后端真实运行）。

## 截图
- 01-新建任务表单.png（全部区块：视频源/字幕插槽/勾选/方案/参数/知识库）
- 02-KB多选与优先级.png（KB#1 优先级1 v3、KB#3 优先级2 v1）
- 03-参数校验.png
- 04-提交后跳详情.png（跳转到新任务详情，import running）

## 说明 / 风险
- 截图 02 顶部残留“阅读深度须为非负整数”红字，是步骤3校验后的界面状态，属测试过程，不影响提交（步骤5已 201）。
