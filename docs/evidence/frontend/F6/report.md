# F6 修复：空知识库列表 → 新建任务页白屏

- 分支：feat/frontend-kbfix（off origin/dev 952fc86）
- 环境：真实浏览器 Playwright+Edge；前端 vite dev（5176）与生产构建 vite preview（4176）；后端 dev 源码二进制，**全新空 DATA_DIR**（DB 无任何知识库/方案），PASSWORD=test123。代理 /api→8091。

## 根因（已实证）
- 空 DB 时后端返回 JSON null：GET /knowledge-bases → `null`，GET /profiles?capability=x → `null`（见复现：kbs raw [null]、profiles raw [null]）。
- TaskCreate.vue load() 未兜底：kbs.value=null → 模板 `!kbs.length` / `kbs.length` 触发 `Cannot read properties of null (reading 'length')` → 内容区白屏。ProfilesView 的 `profiles.value.filter` 同样对 null 崩溃。

## 修复
1) TaskCreate.vue load()：profiles 每类与 kbs 统一 Array.isArray 兜底为 []。
2) ProfilesView.vue load()：profiles 结果 Array.isArray 兜底（原 null.filter 会崩）。
3) KnowledgeBasesView.vue：load()/toggleVersions/upload/delVersion 的 kbs 与 kbVersions 结果统一数组兜底。
4) TasksView（已有 ||[]）、TaskDetail（stages 有 ||[]，task 有 v-if）核查安全，无需改。

## 验收（真实浏览器，均 Pass）
| 项 | 场景 | 期望 | 实测 | 结果 |
|----|------|------|------|------|
| 复现 | 空库 /tasks/new（修复前） | 抛 TypeError | PAGEERROR: Cannot read properties of null (reading 'length') | 已复现 |
| A | 空库 /tasks/new（修复后） | 正常、无 TypeError | H2 在、暂无知识库、5 插槽、errors=[] | ✅ |
| B | 空库 /profiles | 正常 | 渲染正常、errors=[] | ✅ |
| C | 空库 /knowledge-bases | 正常 | 渲染正常、errors=[] | ✅ |
| D | 建 1 个 KB 后 /tasks/new | 5 插槽 + KB 可选 | 5 插槽、1 个 KB、errors=[] | ✅ |
| 生产 | 生产构建 + 空库 /tasks/new | 不白屏、无 TypeError | H2/暂无知识库/5 插槽、errors=[] | ✅ |

## 关键原始输出
- 修复前 repro.json errors: ["PAGEERROR: Cannot read properties of null (reading 'length')"]。
- 修复后 console-errors.json: []（空）；steps.json 4 项 pass 全 true；prod-steps.json pass true。
- npm run build：✓ built，exit=0。

## 截图
- 00-空库-before-fix.png（修复前，dev；关键差异在控制台 TypeError，见 repro.json）
- 01-空库-新建页正常.png / 02-空库-配置方案页正常.png / 03-空库-知识库页正常.png / 04-建KB后-新建页插槽正常.png
- 05-生产构建-空库新建页正常.png（生产产物 vite preview + 空库，无白屏）

## 说明
- dev 下修复前后截图肉眼相似（Vue 对单个 v-if 抛错仍渲染其余区域），真正差异是控制台 TypeError 从有到无；生产构建下即原 M3 白屏，现已修复。
- 空 DB 时 /tasks /profiles /knowledge-bases 均可能返回 null，已统一兜底。
