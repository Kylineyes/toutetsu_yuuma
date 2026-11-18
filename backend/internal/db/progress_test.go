package db

import (
	"path/filepath"
	"testing"
)

// helper: 快速构造一个阶段
func mkStage(stage, status string, progress int) TaskStage {
	return TaskStage{Stage: stage, Status: status, Progress: progress}
}

// TestComputeProgress_Empty 无任何阶段时应为 0。
func TestComputeProgress_Empty(t *testing.T) {
	if got := ComputeProgress(nil); got != 0 {
		t.Fatalf("空阶段应为 0，实际 %d", got)
	}
}

// TestComputeProgress_AllDone 全部阶段 done 应为 100。
func TestComputeProgress_AllDone(t *testing.T) {
	stages := []TaskStage{
		mkStage("import", "done", 100),
		mkStage("audio", "done", 100),
		mkStage("asr", "done", 100),
		mkStage("frame", "done", 100),
		mkStage("vision", "done", 100),
		mkStage("fuse", "done", 100),
		mkStage("summary", "done", 100),
	}
	if got := ComputeProgress(stages); got != 100 {
		t.Fatalf("全部 done 应为 100，实际 %d", got)
	}
}

// TestComputeProgress_SkippedCountsDone skipped 阶段按完成计入。
func TestComputeProgress_SkippedCountsDone(t *testing.T) {
	stages := []TaskStage{
		mkStage("import", "done", 100),
		mkStage("audio", "done", 100),
		mkStage("asr", "skipped", 0),
		mkStage("frame", "done", 100),
		mkStage("vision", "done", 100),
		mkStage("fuse", "done", 100),
		mkStage("summary", "done", 100),
	}
	// 全阶段均出现（asr 为 skipped，按完成计入），应为 100
	if got := ComputeProgress(stages); got != 100 {
		t.Fatalf("含 skipped 全完成应为 100，实际 %d", got)
	}
}

// TestComputeProgress_Partial 部分阶段完成时进度应在合理区间，且随阶段推进单调增加。
func TestComputeProgress_Partial(t *testing.T) {
	// 仅 import+audio 完成：权重 10+10=20，总注册权重 100 => 20
	s1 := []TaskStage{
		mkStage("import", "done", 100),
		mkStage("audio", "done", 100),
	}
	if got := ComputeProgress(s1); got != 20 {
		t.Fatalf("import+audio 完成应为 20，实际 %d", got)
	}

	// 加上 asr 正在跑 50%：10+10 + 25*0.5 = 32.5 => 32
	s2 := append(append([]TaskStage{}, s1...), mkStage("asr", "running", 50))
	if got := ComputeProgress(s2); got != 32 {
		t.Fatalf("再加 asr 50%% 应为 32，实际 %d", got)
	}

	// asr 完成：10+10+25=45
	s3 := append(append([]TaskStage{}, s1...), mkStage("asr", "done", 100))
	if got := ComputeProgress(s3); got != 45 {
		t.Fatalf("asr 完成后应为 45，实际 %d", got)
	}

	// 单调性检查
	if !(ComputeProgress(s1) <= ComputeProgress(s2) && ComputeProgress(s2) <= ComputeProgress(s3)) {
		t.Fatalf("进度应单调不减")
	}
}

// TestComputeProgress_Failed 失败阶段贡献 0，但整体进度不归零。
func TestComputeProgress_Failed(t *testing.T) {
	stages := []TaskStage{
		mkStage("import", "done", 100),
		mkStage("audio", "done", 100),
		mkStage("asr", "failed", 0),
	}
	// 权重 10+10 = 20
	if got := ComputeProgress(stages); got != 20 {
		t.Fatalf("失败阶段应为已完成的 20，实际 %d", got)
	}
}

// TestComputeProgress_Clamp 阶段内部 progress 越界应被裁剪。
func TestComputeProgress_Clamp(t *testing.T) {
	stages := []TaskStage{
		mkStage("import", "running", 200), // 越界 -> 100
		mkStage("audio", "running", -50),  // 越界 -> 0
	}
	// 10*1 + 10*0 = 10
	if got := ComputeProgress(stages); got != 10 {
		t.Fatalf("越界裁剪后应为 10，实际 %d", got)
	}
}

// TestRecomputeTaskProgress_Persisted 验证按阶段聚合后确实回写到 tasks.progress。
func TestRecomputeTaskProgress_Persisted(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer d.Close()
	if err := d.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	task := &Task{ID: "t1", Status: "pending", Progress: 0, CostSummary: "{}"}
	if err := d.CreateTask(task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	// 执行三步阶段：import done, audio done, asr running 40%
	if err := d.StartStage("t1", "import"); err != nil { t.Fatal(err) }
	if err := d.FinishStage("t1", "import", "done", "", ""); err != nil { t.Fatal(err) }
	if err := d.StartStage("t1", "audio"); err != nil { t.Fatal(err) }
	if err := d.FinishStage("t1", "audio", "done", "", ""); err != nil { t.Fatal(err) }
	if err := d.StartStage("t1", "asr"); err != nil { t.Fatal(err) }
	if err := d.UpdateStageProgress("t1", "asr", 40, "分片"); err != nil { t.Fatal(err) }

	got, err := d.GetTask("t1")
	if err != nil { t.Fatalf("get task: %v", err) }
	// 期望 10+10 + 25*0.4 = 30
	if got.Progress != 30 {
		t.Fatalf("运行中进度应为 30，实际 %d", got.Progress)
	}

	// 完成 asr 与剩余阶段后应到 100
	for _, s := range []string{"asr"} {
		_ = d.FinishStage("t1", s, "done", "", "")
	}
	for _, s := range []string{"frame", "vision", "fuse", "summary"} {
		_ = d.StartStage("t1", s)
		_ = d.FinishStage("t1", s, "done", "", "")
	}
	if err := d.SetTaskFinished("t1", "done", ""); err != nil { t.Fatal(err) }
	got2, _ := d.GetTask("t1")
	if got2.Progress != 100 {
		t.Fatalf("完成后进度应为 100，实际 %d", got2.Progress)
	}
}

// TestStageWeightsSum100 权重组应规整（正数、总和 100）。
func TestStageWeightsSum100(t *testing.T) {
	sum := 0
	for k, w := range StageWeights {
		if w <= 0 {
			t.Fatalf("阶段 %s 权重非正: %d", k, w)
		}
		sum += w
	}
	if sum != 100 {
		t.Fatalf("权重总和应为 100，实际 %d", sum)
	}
}