package db

import (
	"database/sql"
	"time"
)

// ---------------- profiles ----------------

func (d *DB) CreateProfile(p *Profile) (int64, error) {
	res, err := d.Exec(`INSERT INTO profiles(capability,protocol,name,base_url,api_key,model,temperature,timeout_seconds,extra,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?)`,
		p.Capability, p.Protocol, p.Name, p.BaseURL, p.APIKey, p.Model, nullableFloat(p.Temperature), p.TimeoutSeconds, p.Extra, now())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func nullableFloat(f *float64) any {
	if f == nil {
		return nil
	}
	return *f
}

const profileCols = `id,capability,protocol,name,base_url,api_key,model,temperature,timeout_seconds,extra,created_at`

func scanProfile(sc interface{ Scan(...any) error }) (*Profile, error) {
	var p Profile
	var temp sql.NullFloat64
	err := sc.Scan(&p.ID, &p.Capability, &p.Protocol, &p.Name, &p.BaseURL, &p.APIKey, &p.Model,
		&temp, &p.TimeoutSeconds, &p.Extra, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	if temp.Valid {
		v := temp.Float64
		p.Temperature = &v
	}
	return &p, nil
}

func (d *DB) ListProfiles(capability string) ([]Profile, error) {
	var rows *sql.Rows
	var err error
	if capability != "" {
		rows, err = d.Query(`SELECT `+profileCols+` FROM profiles WHERE capability=? ORDER BY id`, capability)
	} else {
		rows, err = d.Query(`SELECT ` + profileCols + ` FROM profiles ORDER BY id`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Profile
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (d *DB) GetProfile(id int64) (*Profile, error) {
	p, err := scanProfile(d.QueryRow(`SELECT `+profileCols+` FROM profiles WHERE id=?`, id))
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return p, err
}

func (d *DB) UpdateProfile(p *Profile) error {
	_, err := d.Exec(`UPDATE profiles SET capability=?,protocol=?,name=?,base_url=?,api_key=?,model=?,temperature=?,timeout_seconds=?,extra=? WHERE id=?`,
		p.Capability, p.Protocol, p.Name, p.BaseURL, p.APIKey, p.Model, nullableFloat(p.Temperature), p.TimeoutSeconds, p.Extra, p.ID)
	return err
}

func (d *DB) DeleteProfile(id int64) error {
	_, err := d.Exec(`DELETE FROM profiles WHERE id=?`, id)
	return err
}

// ---------------- knowledge bases ----------------

func (d *DB) CreateKnowledgeBase(name string) (int64, error) {
	res, err := d.Exec(`INSERT INTO knowledge_bases(name,latest_version,created_at) VALUES(?,0,?)`, name, now())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) ListKnowledgeBases() ([]KnowledgeBase, error) {
	rows, err := d.Query(`SELECT id,name,latest_version,created_at FROM knowledge_bases ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []KnowledgeBase
	for rows.Next() {
		var k KnowledgeBase
		if err := rows.Scan(&k.ID, &k.Name, &k.LatestVersion, &k.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (d *DB) GetKnowledgeBase(id int64) (*KnowledgeBase, error) {
	var k KnowledgeBase
	err := d.QueryRow(`SELECT id,name,latest_version,created_at FROM knowledge_bases WHERE id=?`, id).
		Scan(&k.ID, &k.Name, &k.LatestVersion, &k.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (d *DB) DeleteKnowledgeBase(id int64) error {
	_, err := d.Exec(`DELETE FROM knowledge_bases WHERE id=?`, id)
	return err
}

// AddKBVersion records a new immutable version snapshot and bumps latest_version.
func (d *DB) AddKBVersion(v *KBVersion) (*KBVersion, error) {
	tx, err := d.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var latest int
	if err := tx.QueryRow(`SELECT latest_version FROM knowledge_bases WHERE id=?`, v.KBID).Scan(&latest); err != nil {
		return nil, err
	}
	v.Version = latest + 1
	v.CreatedAt = now()
	res, err := tx.Exec(`INSERT INTO knowledge_base_versions(kb_id,version,dir_path,file_name,sha256,term_count,size_bytes,created_at)
		VALUES(?,?,?,?,?,?,?,?)`, v.KBID, v.Version, v.DirPath, v.FileName, v.SHA256, v.TermCount, v.SizeBytes, v.CreatedAt)
	if err != nil {
		return nil, err
	}
	v.ID, _ = res.LastInsertId()
	if _, err := tx.Exec(`UPDATE knowledge_bases SET latest_version=? WHERE id=?`, v.Version, v.KBID); err != nil {
		return nil, err
	}
	return v, tx.Commit()
}

const kbVersionCols = `id,kb_id,version,dir_path,file_name,sha256,term_count,size_bytes,created_at`

func scanKBVersion(sc interface{ Scan(...any) error }) (*KBVersion, error) {
	var v KBVersion
	err := sc.Scan(&v.ID, &v.KBID, &v.Version, &v.DirPath, &v.FileName, &v.SHA256, &v.TermCount, &v.SizeBytes, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (d *DB) ListKBVersions(kbID int64) ([]KBVersion, error) {
	rows, err := d.Query(`SELECT `+kbVersionCols+` FROM knowledge_base_versions WHERE kb_id=? ORDER BY version DESC`, kbID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []KBVersion
	for rows.Next() {
		v, err := scanKBVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (d *DB) GetKBVersion(kbID int64, version int) (*KBVersion, error) {
	v, err := scanKBVersion(d.QueryRow(`SELECT `+kbVersionCols+` FROM knowledge_base_versions WHERE kb_id=? AND version=?`, kbID, version))
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return v, err
}

func (d *DB) DeleteKBVersion(kbID int64, version int) error {
	_, err := d.Exec(`DELETE FROM knowledge_base_versions WHERE kb_id=? AND version=?`, kbID, version)
	return err
}

// CountVersionRefs returns how many tasks reference (kbID, version).
func (d *DB) CountVersionRefs(kbID int64, version int) (int, error) {
	var n int
	err := d.QueryRow(`SELECT COUNT(1) FROM task_knowledge_bases WHERE kb_id=? AND version=?`, kbID, version).Scan(&n)
	return n, err
}

// ---------------- tasks ----------------

func (d *DB) CreateTask(t *Task) error {
	_, err := d.Exec(`INSERT INTO tasks(id,status,progress,error,source_type,video_url,video_path,filename,
		summary_preference,screenshot_mode,screenshot_interval,reading_depth,summary_profile_id,asr_profile_id,
		vision_profile_id,reading_profile_id,generate_ai_subtitle_via_asr,generate_vision_subtitle,cost_summary,
		started_at,finished_at,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.Status, t.Progress, t.Error, t.SourceType, t.VideoURL, t.VideoPath, t.Filename,
		t.SummaryPreference, t.ScreenshotMode, t.ScreenshotInterval, t.ReadingDepth, t.SummaryProfileID,
		t.ASRProfileID, t.VisionProfileID, t.ReadingProfileID, boolToInt(t.GenerateAIViaASR), boolToInt(t.GenerateVision),
		t.CostSummary, "", "", now(), now())
	return err
}

func (d *DB) UpdateTaskStatus(id, status string, progress int, errMsg string) error {
	_, err := d.Exec(`UPDATE tasks SET status=?,progress=?,error=?,updated_at=? WHERE id=?`, status, progress, errMsg, now(), id)
	return err
}

func (d *DB) SetTaskStarted(id string) error {
	_, err := d.Exec(`UPDATE tasks SET status='running',started_at=?,updated_at=? WHERE id=?`, now(), now(), id)
	return err
}

func (d *DB) SetTaskFinished(id, status, errMsg string) error {
	// 进度以 task_stages 加权聚合为准：done 强制 100，其它状态保留当前聚合值（失败不清零已完成进度）。
	var progress int
	if status == "done" {
		progress = 100
	} else {
		_ = d.QueryRow("SELECT progress FROM tasks WHERE id=?", id).Scan(&progress)
	}
	_, err := d.Exec(`UPDATE tasks SET status=?,progress=?,error=?,finished_at=?,updated_at=? WHERE id=?`,
		status, progress, errMsg, now(), now(), id)
	return err
}

func (d *DB) UpdateTaskCost(id, costJSON string) error {
	_, err := d.Exec(`UPDATE tasks SET cost_summary=?,updated_at=? WHERE id=?`, costJSON, now(), id)
	return err
}

func (d *DB) UpdateTaskVideo(id, videoPath string) error {
	_, err := d.Exec(`UPDATE tasks SET video_path=?,updated_at=? WHERE id=?`, videoPath, now(), id)
	return err
}

const taskCols = `id,status,progress,error,source_type,video_url,video_path,filename,
	summary_preference,screenshot_mode,screenshot_interval,reading_depth,summary_profile_id,asr_profile_id,
	vision_profile_id,reading_profile_id,generate_ai_subtitle_via_asr,generate_vision_subtitle,cost_summary,
	started_at,finished_at,created_at,updated_at`

func scanTask(sc interface{ Scan(...any) error }) (*Task, error) {
	var t Task
	var g1, g2 int
	err := sc.Scan(&t.ID, &t.Status, &t.Progress, &t.Error, &t.SourceType, &t.VideoURL, &t.VideoPath,
		&t.Filename, &t.SummaryPreference, &t.ScreenshotMode, &t.ScreenshotInterval, &t.ReadingDepth,
		&t.SummaryProfileID, &t.ASRProfileID, &t.VisionProfileID, &t.ReadingProfileID, &g1, &g2, &t.CostSummary,
		&t.StartedAt, &t.FinishedAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	t.GenerateAIViaASR = g1 != 0
	t.GenerateVision = g2 != 0
	return &t, nil
}

func (d *DB) GetTask(id string) (*Task, error) {
	t, err := scanTask(d.QueryRow(`SELECT `+taskCols+` FROM tasks WHERE id=?`, id))
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return t, err
}

func (d *DB) ListTasks() ([]Task, error) {
	rows, err := d.Query(`SELECT ` + taskCols + ` FROM tasks ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (d *DB) DeleteTask(id string) error {
	d.Exec(`DELETE FROM task_subtitles WHERE task_id=?`, id)
	d.Exec(`DELETE FROM task_knowledge_bases WHERE task_id=?`, id)
	d.Exec(`DELETE FROM task_stages WHERE task_id=?`, id)
	_, err := d.Exec(`DELETE FROM tasks WHERE id=?`, id)
	return err
}

// ---------------- task stages ----------------

// EnsureStage inserts a pending stage row if absent.
func (d *DB) EnsureStage(taskID, stage string) error {
	_, err := d.Exec(`INSERT OR IGNORE INTO task_stages(task_id,stage,status,progress) VALUES(?,?,'pending',0)`, taskID, stage)
	return err
}

// StartStage marks a stage running and records its start time.
func (d *DB) StartStage(taskID, stage string) error {
	if err := d.EnsureStage(taskID, stage); err != nil {
		return err
	}
	_, err := d.Exec(`UPDATE task_stages SET status='running',started_at=?,progress=0,error='' WHERE task_id=? AND stage=?`, now(), taskID, stage)
	if err != nil {
		return err
	}
	// 阶段启动后重新聚合任务整体进度。
	_, _ = d.RecomputeTaskProgress(taskID)
	return nil
}

// UpdateStageProgress updates a stage's progress/message.
func (d *DB) UpdateStageProgress(taskID, stage string, progress int, message string) error {
	_, err := d.Exec(`UPDATE task_stages SET progress=?,message=? WHERE task_id=? AND stage=?`, progress, message, taskID, stage)
	if err != nil {
		return err
	}
	// 阶段内部进度变化也会影响整体进度，实时回写。
	_, _ = d.RecomputeTaskProgress(taskID)
	return nil
}

// FinishStage marks a stage done/failed/skipped and records duration.
func (d *DB) FinishStage(taskID, stage, status, message, errMsg string) error {
	if err := d.EnsureStage(taskID, stage); err != nil {
		return err
	}
	var started string
	_ = d.QueryRow(`SELECT started_at FROM task_stages WHERE task_id=? AND stage=?`, taskID, stage).Scan(&started)
	var dur int64
	if started != "" {
		if st, err := time.Parse(time.RFC3339, started); err == nil {
			dur = time.Since(st).Milliseconds()
		}
	}
	progress := 100
	if status == "failed" {
		progress = 0
	}
	_, err := d.Exec(`UPDATE task_stages SET status=?,progress=?,finished_at=?,duration_ms=?,message=?,error=? WHERE task_id=? AND stage=?`,
		status, progress, now(), dur, message, errMsg, taskID, stage)
	if err != nil {
		return err
	}
	// 阶段完成后按权重聚合回写任务整体进度。
	_, _ = d.RecomputeTaskProgress(taskID)
	return nil
}

func (d *DB) ListStages(taskID string) ([]TaskStage, error) {
	rows, err := d.Query(`SELECT id,task_id,stage,status,progress,started_at,finished_at,duration_ms,message,error FROM task_stages WHERE task_id=? ORDER BY id`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TaskStage
	for rows.Next() {
		var s TaskStage
		if err := rows.Scan(&s.ID, &s.TaskID, &s.Stage, &s.Status, &s.Progress, &s.StartedAt, &s.FinishedAt, &s.DurationMs, &s.Message, &s.Error); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ---------------- task subtitles ----------------

func (d *DB) AddTaskSubtitle(s *TaskSubtitle) error {
	_, err := d.Exec(`INSERT INTO task_subtitles(task_id,kind,orig_name,stored_path,generated) VALUES(?,?,?,?,?)`,
		s.TaskID, s.Kind, s.OrigName, s.StoredPath, boolToInt(s.Generated))
	return err
}

func (d *DB) ListTaskSubtitles(taskID string) ([]TaskSubtitle, error) {
	rows, err := d.Query(`SELECT id,task_id,kind,orig_name,stored_path,generated FROM task_subtitles WHERE task_id=? ORDER BY id`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TaskSubtitle
	for rows.Next() {
		var s TaskSubtitle
		var g int
		if err := rows.Scan(&s.ID, &s.TaskID, &s.Kind, &s.OrigName, &s.StoredPath, &g); err != nil {
			return nil, err
		}
		s.Generated = g != 0
		out = append(out, s)
	}
	return out, rows.Err()
}

// ---------------- task knowledge bases ----------------

func (d *DB) AddTaskKB(tk *TaskKB, taskID string) error {
	_, err := d.Exec(`INSERT INTO task_knowledge_bases(task_id,kb_id,version,priority) VALUES(?,?,?,?)`,
		taskID, tk.KBID, tk.Version, tk.Priority)
	return err
}

func (d *DB) ListTaskKBs(taskID string) ([]TaskKB, error) {
	rows, err := d.Query(`SELECT kb_id,version,priority FROM task_knowledge_bases WHERE task_id=? ORDER BY priority ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TaskKB
	for rows.Next() {
		var tk TaskKB
		if err := rows.Scan(&tk.KBID, &tk.Version, &tk.Priority); err != nil {
			return nil, err
		}
		out = append(out, tk)
	}
	return out, rows.Err()
}

// DeleteTaskStages clears all stage rows for a task (full reset).
func (d *DB) DeleteTaskStages(taskID string) error {
	_, err := d.Exec(`DELETE FROM task_stages WHERE task_id=?`, taskID)
	return err
}

// StageStatus 返回某阶段当前状态；不存在返回空字符串。
func (d *DB) StageStatus(taskID, stage string) (string, error) {
	var st string
	err := d.QueryRow(`SELECT status FROM task_stages WHERE task_id=? AND stage=?`, taskID, stage).Scan(&st)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return st, err
}

// ResetNonDoneStages 删除除 done/skipped 之外的阶段记录，用于断点续跑：保留已完成阶段以便跳过。
// 返回被保留（已完成）的阶段名列表，便于日志/证据。
func (d *DB) ResetNonDoneStages(taskID string) ([]string, error) {
	rows, err := d.Query(`SELECT stage,status FROM task_stages WHERE task_id=? ORDER BY id`, taskID)
	if err != nil {
		return nil, err
	}
	var kept []string
	var toDelete []string
	for rows.Next() {
		var stage, status string
		if err := rows.Scan(&stage, &status); err != nil {
			rows.Close()
			return nil, err
		}
		if status == "done" || status == "skipped" {
			kept = append(kept, stage)
		} else {
			toDelete = append(toDelete, stage)
		}
	}
	rows.Close()
	for _, stage := range toDelete {
		if _, err := d.Exec(`DELETE FROM task_stages WHERE task_id=? AND stage=?`, taskID, stage); err != nil {
			return nil, err
		}
	}
	return kept, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
