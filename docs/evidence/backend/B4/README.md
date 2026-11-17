# B4 断点续跑（重跑跳过已完成阶段）

分支：feat/backend-resume（基于 origin/dev cea0eeb，含 B5/B6 基线）

## 变更
### DB（backend/internal/db/repo.go）
- 新增 `StageStatus(taskID, stage)`：查询阶段状态（不存在返回空）。
- 新增 `ResetNonDoneStages(taskID)`：删除非 done/skipped 的阶段记录，**保留已完成阶段**，返回保留列表。用于续跑重置。

### API（backend/internal/api/api_tasks.go retryTask）
- 原逻辑 `DeleteTaskStages`（全清）改为 `ResetNonDoneStages`：只重置失败/未完成阶段。
- 响应增加 `resumed_from`（本次续跑保留的阶段列表）。

### Pipeline（backend/internal/pipeline/pipeline.go Run）
- 每个阶段开始前用 `isDone(stage)` 判断：已完成则跳过执行并从磁盘恢复产物：
  - import：跳过下载/复用 video_path。
  - audio：跳过抽取；若 audio.wav 中间文件已被清理则重跑抽取。
  - asr：从 asr.srt 恢复 asrCues，不再调用转录。
  - frame：跳过抽帧；若后续 vision/回看需要则幂等重抽（覆盖 frames 目录）。
  - vision：从 vision.srt 恢复 visionCues，不再调用识图。
  - fuse：从 fused.srt 恢复 unifiedText，不再融合。
  - summary：跳过生成，保留既有 summary.md/knowledge.md。
- 每次跳过写 `[pipeline] task=... 跳过 X（已完成）` 日志。

## 单元测试（3 用例全绿）
命令：`cd backend && go test ./internal/db/ -v -run "StageStatus|ResetNonDone"`
- TestStageStatus：状态查询（存在/不存在）。
- TestResetNonDoneStages：保留 done、删除 failed/pending。
- TestResetNonDoneStages_SkippedKept：skipped 也保留。
原始输出见 `unittest_output.txt`。

## 端到端续跑验证（真实运行 PORT=18084，真实 openai_chat）
1. 创建任务（不选 summary profile）→ 首次运行：import/audio/fuse **done**（634/906/920ms），summary **failed**（未选配置方案），progress=30。
2. 将 summary_profile_id 置 1 后 POST /tasks/{id}/retry → 响应 `{"resumed_from":["import","audio","fuse"],"status":"queued"}`。
3. 重跑后：import/audio/fuse 的 started_at/finished_at/duration_ms **与首次完全相同**（stages id 1/3/5 未变，证明未重跑），仅 summary 重新执行（新 id 9，10044ms）；任务终态 **done progress=100**。
4. 服务端日志明确：
   `跳过 import（已完成）` / `跳过 audio（已完成）` / `跳过 fuse（已完成），从 fused.srt 恢复`
证据：`resume_server_log.txt`、`task_after_retry.json`、`stages_after_retry.json`。

## 结论
重跑跳过已完成阶段，仅执行失败/未完成阶段；日志与阶段时间戳双重证明跳过；DB 单测覆盖重置逻辑。
