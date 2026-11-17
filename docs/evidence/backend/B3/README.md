# B3 进度聚合（tasks.progress 由 task_stages 加权计算）

## 变更
- 新增 `backend/internal/db/progress.go`：`StageWeights`（import10/audio10/asr25/frame10/vision15/fuse10/summary20，总和 100）、`ComputeProgress(stages)`、`RecomputeTaskProgress(taskID)`。
- `backend/internal/db/repo.go`：`StartStage` / `UpdateStageProgress` / `FinishStage` 在写阶段后调用 `RecomputeTaskProgress` 回写 `tasks.progress`；`SetTaskFinished` 在 done 时置 100，否则保留聚合值（失败不再粗暴清零）。
- `backend/internal/pipeline/pipeline.go`：移除 5 处硬编码里程碑进度（15/22/45/55/65），改由阶段聚合自动维护。

## 聚合规则
- 分母为全部注册阶段权重和（100）；未创建阶段按 0 计入。
- done / skipped 计 100%；failed 计 0%；running / pending 按其 `progress`(0..100) 折算。
- 结果 clamp 到 0..100。

## 单元测试（全绿）
命令：`cd backend && go test ./internal/db/ -v`
原始输出见 `unittest_output.txt`，用例：
- TestComputeProgress_Empty / AllDone / SkippedCountsDone / Partial / Failed / Clamp
- TestRecomputeTaskProgress_Persisted（真实 SQLite：StartStage/FinishStage/UpdateStageProgress 后回写进度 30 → 100）
- TestStageWeightsSum100

## 运行中接口验证（真实启动应用）
命令：
```
set DATA_DIR/TMP_DIR/PORT=18080; go build -o server.exe ./cmd/server; .\server.exe
curl -s http://127.0.0.1:18080/api/v1/health   # {"status":"ok"}
curl -s -X POST -F "video_file=@_tiny.mp4" http://127.0.0.1:18080/api/v1/tasks
curl -s http://127.0.0.1:18080/api/v1/tasks/$TID
curl -s http://127.0.0.1:18080/api/v1/tasks/$TID/stages
```
观测（真实任务，故意不选 summary profile 触发在 summary 阶段失败）：
- import done(317ms) / audio done(1518ms) / fuse done(537ms) / summary failed(559ms，"未选择配置方案")
- `GET /tasks/{id}` 返回 `progress=30`：恰为 import(10)+audio(10)+fuse(10)=30 的加权结果，符合预期（不再硬编码）。
- 原始 JSON 见 `live_task.json`、`live_stages.json`。

## 结论
tasks.progress 已由 task_stages 加权聚合驱动，替换硬编码；单测覆盖各状态与单调性，运行中接口值合理（30%）。
