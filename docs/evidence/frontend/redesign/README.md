# 前端重构视觉验收（F-R1）

- 分支：feat/frontend-redesign（off origin/dev 6a9bfe4）
- 范围：明暗主题、字号/配色 token、连续曲率、临界阻尼动效、布局重构
- 规范：[docs/design/前端设计规范.md](../../../design/前端设计规范.md)

## 方法

1. `cd frontend && npm install && npm run build`（产物 dist/，构建通过）。
2. `npx vite preview --port 4173 --strictPort`（生产产物，无 dev 监听器，规避 Windows 下编辑器临时文件导致的 EBUSY）。
3. `node capture.mjs`：Playwright 1.65 + 系统 Edge（`channel: msedge`），1440×900，`page.route('**/api/v1/**')` 注入确定性假数据，逐页截图明/暗两套。

> 复现：任意目录 `npm i playwright` 后，`OUT_DIR=<本目录> node capture.mjs`（脚本以 OUT_DIR 覆盖输出位置，缺省为脚本所在目录）。

## 结果

| 项 | 期望 | 实测 | 结论 |
|---|---|---|---|
| 构建 | 成功 | vite build 通过，CSS 22.95 kB / JS 137.76 kB（gzip 5.33/51.29） | 通过 |
| 明色主题 | data-theme=light，canvas #f6f7f9 | 6 页全部 light，body 背景 rgb(246,247,249) | 通过 |
| 暗色主题 | data-theme=dark，canvas #11151c | 6 页全部 dark，body 背景 rgb(17,21,28) | 通过 |
| 切换持久化 | 点击写 ty-theme，刷新保持 | light（无存储）→ 点击 dark → 刷新 dark → 切回 light，存储同步 | 通过 |
| 系统跟随 | 无 ty-theme 时跟随 prefers-color-scheme | 初始无存储时取 light（colorScheme=light 环境） | 通过 |
| 控制台 | 无 error | 12 次页面加载 + 切换，consoleErrors=[] | 通过 |
| 窄屏 | 390px 无横向溢出 | documentElement.scrollWidth=390=clientWidth | 通过（修复后） |
| 表格 | 容器内横向滚动 | 表格 799px 在 356px 容器内滚动，页面不溢出 | 通过 |

## 评审中修掉的 2 个缺陷

| 缺陷 | 表现 | 修复 |
|---|---|---|
| 侧栏品牌被当作普通链接 | `.brand` 是 RouterLink，继承 `a` 的下划线与链接色，暗色下发蓝 | `.brand` 显式 `color: sidebar-text; text-decoration: none` |
| 窄屏页面横向溢出 709px | 表格内 `.sr-only` 绝对定位无定位祖先，静态位置落在 708px 处撑开文档 | `.sr-only` 增加 `left/top: 0` 与 `clip-path`；`.table-scroll` 设 `position: relative` |

另修一处语义问题：任务列表空态的 `RouterLink` 内嵌 `UiButton`（a 套 button），改为 `UiButton :to`。

## 截图

| 文件 | 页面 |
|---|---|
| light/tasks.png、dark/tasks.png | 任务列表（含 4 种状态徽标与进度条） |
| light/tasks-new.png、dark/tasks-new.png | 新建任务（5 段表单 + KB 多选） |
| light/task-detail.png、dark/task-detail.png | 任务详情（概览/阶段/产物/成本/正文） |
| light/profiles.png、dark/profiles.png | 配置方案（表单 + 四类能力表） |
| light/knowledge-bases.png、dark/knowledge-bases.png | 知识库（创建/搜索/版本） |
| light/login.png、dark/login.png | 登录页（独立布局 + 浮动主题按钮） |
| mobile-tasks.png | 390×844 窄屏 |
| toggle-after-click.png | 点击切换为暗色后的任务页 |
| report.json | 上述断言的机器可读结果 |

## 未覆盖

- 真实后端联调：本次用路由 mock 固定数据，未连真实 API（数据形状依据 `frontend/src/api.js` 与 `docs/api.md`）。
- 浏览器矩阵：仅 Edge（Chromium 系）。`corner-shape: squircle` 不支持时按规范降级为正圆角，未在 Firefox/Safari 实测。
- 动效观感：仅校验曲线 token 与 `prefers-reduced-motion` 规则存在，未做逐帧比对。
