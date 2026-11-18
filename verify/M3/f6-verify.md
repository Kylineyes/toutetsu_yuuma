# F6/B7 修复复验（空库端到端）

- 基线：dev `e606185`（含 `fix(api): 空列表统一返回[]` 0dd618e 与 `fix(frontend): F6 空知识库白屏兜底` 6688591）
- 环境：全新空 DATA_DIR 启后端(PASSWORD=test123, PORT18090) + dev 最新构建 dist 静态托管反代 /api。

## 1. B7 空列表返回 []（curl 原始 body，全部为空数组非 null）
见 `f6-empty-lists-curl.txt`：
- GET /knowledge-bases => []
- GET /profiles => []
- GET /profiles?capability=summary/asr/vision/reading => []
- GET /tasks => []
（B7 在 writeJSON 中新增 normalizeNilSlice，对 nil 切片统一编码为 []。）

## 2. F6 浏览器空库无白屏（CDP 真实浏览器，7/7 PASS）
- /tasks/new：h2=新建任务、5 个 file 插槽齐全、显示“暂无知识库”、**无 console TypeError/error**。截图 F6-tasks-new-empty-ok.png
- /profiles：h2=配置方案（Profile），内容正常，无异常。截图 F6-profiles-empty-ok.png
- /knowledge-bases：h2=知识库，正常，无异常。截图 F6-kbs-empty-ok.png
- 建 1 个 KB 后回 /tasks/new：正常列出“F6验证库”可选；勾选（无版本）提示“该知识库尚无版本，无法选择”，页面不崩溃。截图 F6-tasks-new-with-kb.png

## 结论
- 之前 M3 发现的“空知识库列表白屏”缺陷**已修复**（前端数组兜底 + 后端空集返回[]，双保险）。
- 空库场景端到端正常，无白屏、无异常。
