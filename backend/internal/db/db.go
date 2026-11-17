package db

import (
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps *sql.DB with project-specific helpers.
type DB struct {
	*sql.DB
}

// Open opens (and creates if needed) the SQLite database at path.
func Open(path string) (*DB, error) {
	sqldb, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	sqldb.SetMaxOpenConns(1)
	if err := sqldb.Ping(); err != nil {
		return nil, err
	}
	return &DB{sqldb}, nil
}

// Migrate creates all tables and indexes if they do not exist.
func (d *DB) Migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS profiles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			capability TEXT NOT NULL,
			protocol TEXT NOT NULL DEFAULT 'openai_chat',
			name TEXT NOT NULL,
			base_url TEXT NOT NULL,
			api_key TEXT NOT NULL,
			model TEXT NOT NULL,
			temperature REAL,
			timeout_seconds INTEGER NOT NULL DEFAULT 180,
			extra TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS knowledge_bases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			latest_version INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS knowledge_base_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			kb_id INTEGER NOT NULL,
			version INTEGER NOT NULL,
			dir_path TEXT NOT NULL,
			file_name TEXT NOT NULL,
			sha256 TEXT NOT NULL DEFAULT '',
			term_count INTEGER NOT NULL DEFAULT 0,
			size_bytes INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			UNIQUE(kb_id, version)
		)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			status TEXT NOT NULL,
			progress INTEGER NOT NULL DEFAULT 0,
			error TEXT NOT NULL DEFAULT '',
			source_type TEXT NOT NULL DEFAULT 'file',
			video_url TEXT NOT NULL DEFAULT '',
			video_path TEXT NOT NULL DEFAULT '',
			filename TEXT NOT NULL DEFAULT '',
			summary_preference TEXT NOT NULL DEFAULT '',
			screenshot_mode TEXT NOT NULL DEFAULT 'simple',
			screenshot_interval INTEGER NOT NULL DEFAULT 60,
			reading_depth INTEGER NOT NULL DEFAULT 2,
			summary_profile_id INTEGER NOT NULL DEFAULT 0,
			asr_profile_id INTEGER NOT NULL DEFAULT 0,
			vision_profile_id INTEGER NOT NULL DEFAULT 0,
			reading_profile_id INTEGER NOT NULL DEFAULT 0,
			generate_ai_subtitle_via_asr INTEGER NOT NULL DEFAULT 0,
			generate_vision_subtitle INTEGER NOT NULL DEFAULT 0,
			cost_summary TEXT NOT NULL DEFAULT '{}',
			started_at TEXT NOT NULL DEFAULT '',
			finished_at TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS task_stages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id TEXT NOT NULL,
			stage TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			progress INTEGER NOT NULL DEFAULT 0,
			started_at TEXT NOT NULL DEFAULT '',
			finished_at TEXT NOT NULL DEFAULT '',
			duration_ms INTEGER NOT NULL DEFAULT 0,
			message TEXT NOT NULL DEFAULT '',
			error TEXT NOT NULL DEFAULT '',
			UNIQUE(task_id, stage)
		)`,
		`CREATE TABLE IF NOT EXISTS task_subtitles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			orig_name TEXT NOT NULL,
			stored_path TEXT NOT NULL,
			generated INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS task_knowledge_bases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id TEXT NOT NULL,
			kb_id INTEGER NOT NULL,
			version INTEGER NOT NULL,
			priority INTEGER NOT NULL DEFAULT 0
		)`,
		// indexes
		`CREATE INDEX IF NOT EXISTS idx_task_subtitles_task ON task_subtitles(task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_task_kb_task ON task_knowledge_bases(task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_task_stages_task ON task_stages(task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_kb_versions_kb ON knowledge_base_versions(kb_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)`,
	}
	for _, s := range stmts {
		if _, err := d.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

func now() string { return time.Now().Format(time.RFC3339) }

// ---------------- models ----------------

type Profile struct {
	ID             int64    `json:"id"`
	Capability     string   `json:"capability"`
	Protocol       string   `json:"protocol"`
	Name           string   `json:"name"`
	BaseURL        string   `json:"base_url"`
	APIKey         string   `json:"api_key,omitempty"`
	Model          string   `json:"model"`
	Temperature    *float64 `json:"temperature"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	Extra          string   `json:"extra,omitempty"`
	CreatedAt      string   `json:"created_at"`
}

type KnowledgeBase struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	LatestVersion int    `json:"latest_version"`
	CreatedAt     string `json:"created_at"`
}

type KBVersion struct {
	ID        int64  `json:"id"`
	KBID      int64  `json:"kb_id"`
	Version   int    `json:"version"`
	DirPath   string `json:"dir_path"`
	FileName  string `json:"file_name"`
	SHA256    string `json:"sha256"`
	TermCount int    `json:"term_count"`
	SizeBytes int64  `json:"size_bytes"`
	CreatedAt string `json:"created_at"`
}

type Task struct {
	ID                 string `json:"id"`
	Status             string `json:"status"`
	Progress           int    `json:"progress"`
	Error              string `json:"error"`
	SourceType         string `json:"source_type"`
	VideoURL           string `json:"video_url"`
	VideoPath          string `json:"video_path"`
	Filename           string `json:"filename"`
	SummaryPreference  string `json:"summary_preference"`
	ScreenshotMode     string `json:"screenshot_mode"`
	ScreenshotInterval int    `json:"screenshot_interval"`
	ReadingDepth       int    `json:"reading_depth"`
	SummaryProfileID   int64  `json:"summary_profile_id"`
	ASRProfileID       int64  `json:"asr_profile_id"`
	VisionProfileID    int64  `json:"vision_profile_id"`
	ReadingProfileID   int64  `json:"reading_profile_id"`
	GenerateAIViaASR   bool   `json:"generate_ai_subtitle_via_asr"`
	GenerateVision     bool   `json:"generate_vision_subtitle"`
	CostSummary        string `json:"cost_summary"`
	StartedAt          string `json:"started_at"`
	FinishedAt         string `json:"finished_at"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

type TaskStage struct {
	ID         int64  `json:"id"`
	TaskID     string `json:"task_id"`
	Stage      string `json:"stage"`
	Status     string `json:"status"`
	Progress   int    `json:"progress"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
	DurationMs int64  `json:"duration_ms"`
	Message    string `json:"message"`
	Error      string `json:"error"`
}

type TaskSubtitle struct {
	ID         int64  `json:"id"`
	TaskID     string `json:"task_id"`
	Kind       string `json:"kind"`
	OrigName   string `json:"orig_name"`
	StoredPath string `json:"stored_path"`
	Generated  bool   `json:"generated"`
}

type TaskKB struct {
	KBID     int64 `json:"kb_id"`
	Version  int   `json:"version"`
	Priority int   `json:"priority"`
}

var ErrNotFound = errors.New("not found")
