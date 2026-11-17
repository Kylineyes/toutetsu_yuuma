package db

import "strings"

// stageOrder 定义流水线阶段的展示与聚合顺序（与 pipeline 执行顺序一致）。
var stageOrder = []string{"import", "audio", "asr", "frame", "vision", "fuse", "summary"}

// StageWeights 定义各阶段在整体进度中的权重，之和为 100。
// 权重反映相对耗时与重要性，可在不改动其它逻辑的前提下集中调整。
var StageWeights = map[string]int{
	"import":  10,
	"audio":   10,
	"asr":     25,
	"frame":   10,
	"vision":  15,
	"fuse":    10,
	"summary": 20,
}

// orderedStageWeights 返回按 stageOrder 排列的 (阶段, 权重)，仅包含注册权重且各阶段仅出现一次。
func orderedStageWeights() ([]string, []int) {
	seen := map[string]bool{}
	var stages []string
	var weights []int
	for _, s := range stageOrder {
		if seen[s] {
			continue
		}
		w, ok := StageWeights[s]
		if !ok || w <= 0 {
			continue
		}
		seen[s] = true
		stages = append(stages, s)
		weights = append(weights, w)
	}
	return stages, weights
}

// ComputeProgress 依据已存在的 task_stages 计算任务整体进度（0..100）。
//
// 规则：
//   - 权重归一化到"任务实际存在（或已注册）的阶段"上，避免某个任务缺失阶段时进度永远到不了 100。
//   - skipped 的阶段视为已处理完毕，按 100% 计入（该阶段本就要跳过）。
//   - done 按 100% 计入；failed 按 0% 计入；running/pending 按该阶段自身的 progress(0..100) 折算。
//   - 无任何阶段数据时返回 0。
func ComputeProgress(stages []TaskStage) int {
	if len(stages) == 0 {
		return 0
	}
	byStage := map[string]TaskStage{}
	for _, s := range stages {
		byStage[s.Stage] = s
	}
	regStages, regWeights := orderedStageWeights()
	// 分母为全部注册阶段权重之和（固定 100）。未创建的阶段贡献 0，因此运行中进度是
	// "已完成权重 / 全量权重"；对于跳过某阶段的任务，该阶段会被标记为 skipped（按完成计入），
	// 从而不影响最终到达 100。
	totalWeight := 0
	acc := 0.0
	for i, stage := range regStages {
		w := regWeights[i]
		totalWeight += w
		st, ok := byStage[stage]
		if !ok {
			continue
		}
		var frac float64
		switch st.Status {
		case "done", "skipped":
			frac = 1
		case "failed":
			frac = 0
		default: // pending / running
			frac = float64(clampInt(st.Progress, 0, 100)) / 100
		}
		acc += float64(w) * frac
	}
	if totalWeight == 0 {
		return 0
	}
	p := int(acc / float64(totalWeight) * 100)
	return clampInt(p, 0, 100)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// RecomputeTaskProgress 读取任务的阶段并回写 tasks.progress；返回计算后的进度。
func (d *DB) RecomputeTaskProgress(taskID string) (int, error) {
	stages, err := d.ListStages(taskID)
	if err != nil {
		return 0, err
	}
	p := ComputeProgress(stages)
	if _, err := d.Exec("UPDATE tasks SET progress=?,updated_at=? WHERE id=?", p, now(), taskID); err != nil {
		return 0, err
	}
	return p, nil
}

// 保证 strings 包被使用（兼容未来可能的扩展），避免未使用导入。
var _ = strings.TrimSpace