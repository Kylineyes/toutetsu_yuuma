# V1 附加项：B3 进度聚合真实运行验证

目的：核对创建任务跑到部分阶段后，`GET /tasks/{id}` 的 `progress` 是否按已完成阶段**加权聚合**（而非恒 0 或事后 100）。

## 实现依据（读码，ver_wt @ 0a35186）
- `backend/internal/db/progress.go`：`StageWeights{import:10,audio:10,asr:25,frame:10,vision:15,fuse:10,summary:20}`，`ComputeProgress` 按 `done/skipped=100%`、`failed=0%`、`running/pending=自身progress`，分母为全部注册阶段权重和（100）。
- `StartStage`/`UpdateStageProgress`/`FinishStage`（repo.go:314/328/339）均调用 `RecomputeTaskProgress` 实时回写 `tasks.progress`。

## 运行观测 1（无音轨视频 → audio 失败）
轮询 `GET /tasks/{id}`：
- import=done(100) → 任务 progress=**10**（仅 import 完成，权重 10/100）
- audio=running(0) → progress 维持 **10**
- audio=failed → progress=**10**（import 10 + audio 0）
→ 说明失败阶段按 0 计入、已完成阶段按权重计入，非恒 0。

## 运行观测 2（带音轨视频，无 profile）
- import done (10) + audio done (10) + fuse done (10) → 任务 progress=**30**
- summary failed（未选 profile，profile_id=0）→ 该阶段 0
- 观测到的变化点：running progress=10 →（阶段推进）failed progress=30
最终 stages：import done / audio done / fuse done / summary failed(未选择配置方案)
→ **progress 随已完成阶段按权重单调上升（10→30），既非恒 0 也非事后直接 100。B3 聚合行为在真实运行中成立。**

> 附注：任务在 summary 阶段失败仅因验证时未配置真实 AI profile（profile_id=0），属预期；ASR/vision/frame 因无对应 profile 跳过。真实 AI 端到端需真实 Key，超出 V1 契约复检范围（如实说明）。
