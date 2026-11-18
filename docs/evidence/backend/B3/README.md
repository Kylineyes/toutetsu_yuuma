# B3 进度聚合（tasks.progress 由 task_stages 加权计算）

## 修复记录（打回后）
初次提交漏了关键一步：`StartStage` / `UpdateStageProgress` / `FinishStage` 未调用 `RecomputeTaskProgress`，
导致运行中 tasks.progress 停在旧值（测试取到 0）。本次修复：
- `StartStage`：更新 task_stages(running) 成功后 `_, _ = d.RecomputeTaskProgress(taskID)`
- `UpdateStageProgress`：更新 progress/message 成功后重算
- `FinishStage`：更新 done/failed/skipped 成功后重算
- `SetTaskFinished`：done 强制 100，其它状态保留当前聚合值（失败不再清零）

## 变更文件
- 新增 `backend/internal/db/progress.go`：`StageWeights`(import10/audio10/asr25/frame10/vision15/fuse10/summary20=100)、`ComputeProgress`、`RecomputeTaskProgress`
- `backend/internal/db/repo.go`：上述 4 个函数触发/利用聚合
- `backend/internal/pipeline/pipeline.go`：删除 5 处硬编码里程碑进度（15/22/45/55/65）

## 聚合规则
- 分母为全部注册阶段权重和(100)；未创建阶段按 0 计入。
- done/skipped=100%；failed=0%；running/pending 按自身 progress(0..100) 折算；结果 clamp 0..100。

## 单元测试（8/8 全绿，连跑 3 次稳定）
命令：`cd backend && go test ./internal/db/ -v -count=1`
原始完整输出见 `unittest_output.txt`（8 个用例全部 PASS，含 TestRecomputeTaskProgress_Persisted：真实 SQLite 下 StartStage/FinishStage/UpdateStageProgress 后回写进度 30→100）。

## 运行中接口验证（真实启动应用，本 worktree）
命令：
```
PORT=18083 DATA_DIR=.../data TMP_DIR=.../tmp ./server_b3.exe
curl -s -X POST -F "video_file=@tiny.mp4" http://127.0.0.1:18083/api/v1/tasks
curl -s http://127.0.0.1:18083/api/v1/tasks/$TID           # progress=30
curl -s http://127.0.0.1:18083/api/v1/tasks/$TID/stages
```
观测（真实任务，故意不选 summary profile 在 summary 阶段失败）：
- import done(226ms) / audio done(525ms) / fuse done(582ms) / summary failed(612ms)
- `GET /tasks/{id}` → status=failed, **progress=30**（=未完 summary 时已完成 import10+audio10+fuse10 的加权值；非事后补 100，因失败不清零）
- 原始 JSON 见 `live_task.json` / `live_stages.json`

## 结论
tasks.progress 已由 task_stages 加权聚合驱动，替换硬编码；单测 8/8 稳定全绿；运行中接口值真实反映完成比例（30%）。
