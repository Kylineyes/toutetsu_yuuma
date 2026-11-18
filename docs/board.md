# 任务看板（Lead 维护）

> 说明：本会话未启用 Agent Teams，共享看板改为本文件登记。Lead 只做调度/裁定/汇总，不写业务代码。
> 基线：dev=a98f87e，工作树干净。自检：go build ✅、npm run build ✅（Lead 已跑，2026-11-18）。
> 凡「完成」必须附可复现证据（命令+原始输出+产物路径）；Verification 独立复跑，不采信自述；无证据=未完成，打回。

## 状态图例：⬜未开始 / 🔵进行中 / 🟡待验证 / ✅通过 / ❌打回

## 工作项

| ID | 标题 | 分支(off dev) | 写作用域 | 依赖 | 状态 | 证据路径 | 备注 |
|----|------|--------------|----------|------|------|----------|------|
| B1 | 真实 AI e2e 打通 | feat/backend-e2e | backend/,docs/evidence/backend/ | - | ✅ | docs/evidence/backend/B1/ | 接受(ASR降级)；summary/knowledge/fused 真实生成，stages 真实耗时 |
| B2 | 协议实测修正 | feat/backend-protocols | backend/,docs/evidence/backend/ | B1 | ✅ | docs/evidence/backend/B2/ | 独立复跑 build/vet=0 test全ok；gemini thought 跳过修复+四协议实测 |
| B3 | progress 聚合 | feat/backend-progress | backend/,docs/evidence/backend/ | - | ✅ | docs/evidence/backend/B3/ | ba2a119 修复复验：独立复跑 8/8 PASS 3次稳定，live progress=30 真实 |
| B4 | 断点续跑 | feat/backend-resume | backend/,docs/evidence/backend/ | B1,B3 | ✅ | docs/evidence/backend/B4/ | 独立复跑 build/vet=0 test全ok；isDone 跳过+ResetNonDoneStages，stage id 1/3/5 未重跑 |
| B5 | 单元测试 | feat/backend-tests | backend/,docs/evidence/backend/ | - | ✅ | docs/evidence/backend/B5/ | 独立复跑 32/32 PASS；api/db/pipeline/subtitle 全 ok |
| B6 | 上传安全 | feat/backend-upload-sec | backend/,docs/evidence/backend/ | - | ✅ | docs/evidence/backend/B6/ | 7 安全测试全 PASS；live 413/穿越清洗/清理生效 |
| F1 | 登录联调 | feat/frontend-login | frontend/,docs/evidence/frontend/ | - | ✅ | docs/evidence/frontend/F1/ | 真浏览器 6 步 pass |
| F2 | 任务详情+D1 | feat/frontend-detail | frontend/,docs/evidence/frontend/ | F1 | ✅ | docs/evidence/frontend/F2/ | 独立复跑 build=0；D1 删阶段列实证；下载 Cookie/401 |
| F3 | 配置方案 | feat/frontend-profiles | frontend/,docs/evidence/frontend/ | F1 | ✅ | docs/evidence/frontend/F3/ | build=0；协议切换/温度/超时/extra 校验 |
| F4 | 知识库 | feat/frontend-kb | frontend/,docs/evidence/frontend/ | F1 | ✅ | docs/evidence/frontend/F4/ | build=0；版本大小/引用删除400 |
| F5 | 新建任务 | feat/frontend-create | frontend/,docs/evidence/frontend/ | F1 | ✅ | docs/evidence/frontend/F5/ | build=0；多插槽/KB多选/空库null修复 |
| V1 | 契约 curl 复核 | verify/v1-run | verify/ | B1,B3 | ✅ | verify/V1/ | 独立实跑 26/26 PASS(exit0)，dev0a35186；B3 运行观测 progress 10→30 真实 |
| V2 | WebUI 逐页复核 | verify/v2-run | verify/ | F1-F5 | ✅ | verify/V2/ | 独立 32/32 PASS，19 截图；D1/F5空库 实证修复 |
| V3 | 构建复跑 | verify/v3-v4-audit | verify/ | - | 🟡 | verify/V3/ | go/npm PASS；docker 本机无 Docker=未验证(缺口)；M3 本地端到端 8/8 PASS |
| V4 | 契约一致性审计 | verify/v3-v4-audit | verify/ | - | ✅ | verify/V4/ | 主契约一致；D1(中)/D2(低)待修 |
| V5 | 验证报告 | verify/m3-v5 | verify/ | V1-V4 | 🟡 | verify/report.md | 已出总报告；待 F6/B7 修复后复跑更新 |
| F6 | 空KB列表白屏修复 | feat/frontend-kbfix | frontend/,docs/evidence/frontend/ | F5 | ✅ | docs/evidence/frontend/F6/ | 独立复跑 build=0；3 view Array.isArray 兜底；空DB真浏览器无白屏 |
| B7 | 空列表返回[] | feat/backend-emptylist | backend/,docs/evidence/backend/ | - | ✅ | docs/evidence/backend/B7/ | 独立复跑 5/5 PASS+全绿；writeJSON normalizeNilSlice 仅对切片，nil map 语义不变 |

## 里程碑
- M1：B1/B3 → V1 通过  ✅（已完成 2026-11-18，dev=0a35186）
- M2：F1–F5 → V2 通过  ✅（已完成 2026-11-18，dev=9af32cc）
- M3：V3 docker 构建 + 容器端到端  🟡（本机无 Docker：静态审查+本地端到端 8/8 PASS 替代，容器构建未验证缺口）
- M4：B2/B4/B5/B6 收尾 → V1/V3 复跑
- M5：Lead 汇总 + 通知用户 dev→main

## 裁定记录
- [M1/B1] 环境独立复核确认无真实可用 ASR 通道（magpie=openai-responses 仅 chat/vision；DEEPSEEK key 直连 401；网关为 Web UI）。裁定：B1 采用降级 ASR（本地 SRT 作 fused 输入），其余阶段真实执行（真实 openai_chat + 真实 ffmpeg）。要求 md 为真实模型生成、证据内写明 ASR 降级并附 404 实测。ASR 缺失记入最终遗留风险，不计 B1 DoD 失败但须透明。批准于 2026-11-18。
- [模型核对] backend/frontend/verify 均以 magpie/workbuddy/deepseek-v4.1-flash 启动，与主会话一致。
- [工作树冲突裁定] 发现 backend/frontend 共用一个工作目录导致 F1 证据提交 7ced3e0 曾落到 feat/backend-progress。核查 7ced3e0 仅含 docs/evidence/frontend/F1/(10文件,无后端代码)。裁定：feat/backend-progress 复位为 B3 提交 aa4e170（已复跑后 Backend 重新提交）；7ced3e0 保留在 feat/frontend-login。建立隔离 worktree：Backend 独占主仓库目录；Frontend 只用 fe_wt；Verify 用 ver_wt。已 prune 杂散 worktree。
- [M2/F1] F1 证据核查通过并接受 ✅：真实浏览器 Playwright+Edge 6 步全 pass，网络 401(错口令)/200(登录/刷新/退出)齐全，后端为 dev 源码二进制。
- [V3/V4] Verify commit 9219d68 已核：go build/vet/npm build PASS(a98f87e)；docker build 因本机无 Docker 未验证（M3 缺口，需有 Docker 机器/CI）。V4 主契约一致。
- [M1回归] B3 打回：Lead 独立复跑 go test ./internal/db/ -v → TestRecomputeTaskProgress_Persisted FAIL(期望30实际0)，连跑3次稳定。根因定位：repo.go:311-345 StartStage/UpdateStageProgress/FinishStage 未调用 RecomputeTaskProgress，progress 仅 SetTaskFinished 写。Backend 汇报"均触发重算""7用例全绿"与事实不符。已打回要求补齐调用+重跑8/8+重取 live 证据。
- [B3复验] Backend ba2a119 修复：三处 Stage 函数补 RecomputeTaskProgress。Lead 独立复跑 go test ./internal/db/ -v → 8/8 PASS（连跑3次 exit0），go vet/build=0，live_task.json progress=30(status=failed,summary 未选profile)。B3 接受 ✅。
- [M1接受] B1 接受 ✅（ASR 降级已注明）：独立核产物 summary.md(概述+4主题分节带时间戳)/knowledge.md(8卡片+目录)/fused.srt(12条36行)/stages 真实耗时(import783/audio1105/fuse180/summary25316ms)。README 标注 ASR 降级+404 实测。
- [M3/V5] Verify m3-v5(6532b8b)：本地端到端 8/8 PASS(等价拓扑,真实 ffmpeg import/audio)；Docker 静态审查出 A1 无healthcheck/A2 npm install未锁/A4 root/A3 未钉digest(均不阻塞)；docker 构建环境无 Docker 未验证。★新缺陷：空KB列表(DB无KB)时 GET /knowledge-bases 返回 null→/tasks/new 白屏 TypeError。Lead 已独立确认根因(kbs.value=await api.kbs() 无兜底 + 模板 !kbs.length)。裁定：F6 前端兜底 + B7 后端空集返回[] 双修，修后复跑。
- [F6/B7] 空KB白屏双修：B7(0dd618e writeJSON normalizeNilSlice 空切片[]) 独立复跑 5/5+全绿 接受；F6(6688591 TaskCreate/Profiles/KnowledgeBases 三 view Array.isArray 兜底) 独立复跑 build=0 接受。两者已合 dev(57de3f0/e606185) 并推送 e606185，合并后 BE build/test 全绿、FE build=0。
- [偏差裁定] D1(中) TasksView.vue:28 渲染 t.stage 但 db.Task 无 stage→列表阶段列恒空。裁定：修前端（该列改显 status 或删列），交 Frontend 处理。D2(低) 契约 single vs 实现 simple：统一为 simple，由 Lead 回写 plan.md。D3/D4/D5 低风险，D4 记入遗留风险。
