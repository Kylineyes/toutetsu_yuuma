package db

import (
	"path/filepath"
	"testing"
)

// openTestDB 打开一个临时数据库并迁移。
func openTestDB(t *testing.T) *DB {
	t.Helper()
	d, err := Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := d.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

// TestStageStatus 校验阶段状态查询（存在/不存在）。
func TestStageStatus(t *testing.T) {
	d := openTestDB(t)
	_ = d.CreateTask(&Task{ID: "t1", Status: "pending", CostSummary: "{}"})
	_ = d.StartStage("t1", "import")
	_ = d.FinishStage("t1", "import", "done", "", "")
	st, err := d.StageStatus("t1", "import")
	if err != nil || st != "done" {
		t.Fatalf("StageStatus(import) 期望 done，实际 %q err=%v", st, err)
	}
	st2, err := d.StageStatus("t1", "not_exist")
	if err != nil || st2 != "" {
		t.Fatalf("不存在阶段应返回空，实际 %q err=%v", st2, err)
	}
}

// TestResetNonDoneStages 校验续跑重置：保留 done/skipped，删除 failed/pending/running。
func TestResetNonDoneStages(t *testing.T) {
	d := openTestDB(t)
	_ = d.CreateTask(&Task{ID: "t1", Status: "pending", CostSummary: "{}"})
	// import done, audio done, asr failed, fuse pending
	_ = d.StartStage("t1", "import")
	_ = d.FinishStage("t1", "import", "done", "", "")
	_ = d.StartStage("t1", "audio")
	_ = d.FinishStage("t1", "audio", "done", "", "")
	_ = d.StartStage("t1", "asr")
	_ = d.FinishStage("t1", "asr", "failed", "", "boom")
	_ = d.EnsureStage("t1", "fuse") // pending

	kept, err := d.ResetNonDoneStages("t1")
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	// kept 应包含 import, audio
	if len(kept) != 2 {
		t.Fatalf("应保留 2 个已完成阶段，实际 %v", kept)
	}
	// asr/fuse 应被删除
	for _, s := range []string{"asr", "fuse"} {
		st, _ := d.StageStatus("t1", s)
		if st != "" {
			t.Fatalf("阶段 %s 应被重置删除，实际状态 %q", s, st)
		}
	}
	// import/audio 仍 done
	for _, s := range []string{"import", "audio"} {
		st, _ := d.StageStatus("t1", s)
		if st != "done" {
			t.Fatalf("阶段 %s 应保留 done，实际 %q", s, st)
		}
	}
}

// TestResetNonDoneStages_SkippedKept 校验 skipped 阶段也保留。
func TestResetNonDoneStages_SkippedKept(t *testing.T) {
	d := openTestDB(t)
	_ = d.CreateTask(&Task{ID: "t1", Status: "pending", CostSummary: "{}"})
	_ = d.StartStage("t1", "asr")
	_ = d.FinishStage("t1", "asr", "skipped", "无可切分音频", "")
	kept, err := d.ResetNonDoneStages("t1")
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	if len(kept) != 1 || kept[0] != "asr" {
		t.Fatalf("skipped 阶段应保留，实际 %v", kept)
	}
}
