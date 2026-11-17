package db

import (
	"database/sql"
)

// ---------------- profiles ----------------

func (d *DB) CreateProfile(p *Profile) (int64, error) {
	res, err := d.Exec(`INSERT INTO profiles(capability,name,base_url,api_key,model,extra,created_at)
		VALUES(?,?,?,?,?,?,?)`, p.Capability, p.Name, p.BaseURL, p.APIKey, p.Model, p.Extra, now())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) ListProfiles(capability string) ([]Profile, error) {
	var rows *sql.Rows
	var err error
	if capability != "" {
		rows, err = d.Query(`SELECT id,capability,name,base_url,api_key,model,extra,created_at FROM profiles WHERE capability=? ORDER BY id`, capability)
	} else {
		rows, err = d.Query(`SELECT id,capability,name,base_url,api_key,model,extra,created_at FROM profiles ORDER BY id`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Profile
	for rows.Next() {
		var p Profile
		if err := rows.Scan(&p.ID, &p.Capability, &p.Name, &p.BaseURL, &p.APIKey, &p.Model, &p.Extra, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (d *DB) GetProfile(id int64) (*Profile, error) {
	var p Profile
	err := d.QueryRow(`SELECT id,capability,name,base_url,api_key,model,extra,created_at FROM profiles WHERE id=?`, id).
		Scan(&p.ID, &p.Capability, &p.Name, &p.BaseURL, &p.APIKey, &p.Model, &p.Extra, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (d *DB) UpdateProfile(p *Profile) error {
	_, err := d.Exec(`UPDATE profiles SET capability=?,name=?,base_url=?,api_key=?,model=?,extra=? WHERE id=?`,
		p.Capability, p.Name, p.BaseURL, p.APIKey, p.Model, p.Extra, p.ID)
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
	res, err := tx.Exec(`INSERT INTO knowledge_base_versions(kb_id,version,dir_path,file_name,term_count,sha256,created_at)
		VALUES(?,?,?,?,?,?,?)`, v.KBID, v.Version, v.DirPath, v.FileName, v.TermCount, v.SHA256, v.CreatedAt)
	if err != nil {
		return nil, err
	}
	v.ID, _ = res.LastInsertId()
	if _, err := tx.Exec(`UPDATE knowledge_bases SET latest_version=? WHERE id=?`, v.Version, v.KBID); err != nil {
		return nil, err
	}
	return v, tx.Commit()
}

func (d *DB) ListKBVersions(kbID int64) ([]KBVersion, error) {
	rows, err := d.Query(`SELECT id,kb_id,version,dir_path,file_name,term_count,sha256,created_at
		FROM knowledge_base_versions WHERE kb_id=? ORDER BY version DESC`, kbID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []KBVersion
	for rows.Next() {
		var v KBVersion
		if err := rows.Scan(&v.ID, &v.KBID, &v.Version, &v.DirPath, &v.FileName, &v.TermCount, &v.SHA256, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (d *DB) GetKBVersion(kbID int64, version int) (*KBVersion, error) {
	var v KBVersion
	err := d.QueryRow(`SELECT id,kb_id,version,dir_path,file_name,term_count,sha256,created_at
		FROM knowledge_base_versions WHERE kb_id=? AND version=?`, kbID, version).
		Scan(&v.ID, &v.KBID, &v.Version, &v.DirPath, &v.FileName, &v.TermCount, &v.SHA256, &v.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ---------------- tasks ----------------

func (d *DB) CreateTask(t *Task) error {
	_, err := d.Exec(`INSERT INTO tasks(id,status,stage,progress,error,source_type,video_url,video_path,filename,
		summary_preference,screenshot_mode,screenshot_interval,reading_depth,summary_profile_id,asr_profile_id,
		vision_profile_id,reading_profile_id,generate_ai_subtitle_via_asr,generate_vision_subtitle,cost_summary,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.Status, t.Stage, t.Progress, t.Error, t.SourceType, t.VideoURL, t.VideoPath, t.Filename,
		t.SummaryPreference, t.ScreenshotMode, t.ScreenshotInterval, t.ReadingDepth, t.SummaryProfileID,
		t.ASRProfileID, t.VisionProfileID, t.ReadingProfileID, boolToInt(t.GenerateAIViaASR), boolToInt(t.GenerateVision),
		t.CostSummary, now(), now())
	return err
}

func (d *DB) UpdateTaskState(id, status, stage string, progress int, errMsg string) error {
	_, err := d.Exec(`UPDATE tasks SET status=?,stage=?,progress=?,error=?,updated_at=? WHERE id=?`,
		status, stage, progress, errMsg, now(), id)
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

const taskCols = `id,status,stage,progress,error,source_type,video_url,video_path,filename,
	summary_preference,screenshot_mode,screenshot_interval,reading_depth,summary_profile_id,asr_profile_id,
	vision_profile_id,reading_profile_id,generate_ai_subtitle_via_asr,generate_vision_subtitle,cost_summary,created_at,updated_at`

func scanTask(sc interface{ Scan(...any) error }) (*Task, error) {
	var t Task
	var g1, g2 int
	err := sc.Scan(&t.ID, &t.Status, &t.Stage, &t.Progress, &t.Error, &t.SourceType, &t.VideoURL, &t.VideoPath,
		&t.Filename, &t.SummaryPreference, &t.ScreenshotMode, &t.ScreenshotInterval, &t.ReadingDepth,
		&t.SummaryProfileID, &t.ASRProfileID, &t.VisionProfileID, &t.ReadingProfileID, &g1, &g2, &t.CostSummary,
		&t.CreatedAt, &t.UpdatedAt)
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
	_, err := d.Exec(`DELETE FROM tasks WHERE id=?`, id)
	d.Exec(`DELETE FROM task_subtitles WHERE task_id=?`, id)
	d.Exec(`DELETE FROM task_knowledge_bases WHERE task_id=?`, id)
	return err
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

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
