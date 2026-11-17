package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/db"
)

func (a *API) createTask(w http.ResponseWriter, r *http.Request) {
	// 上传总大小上限：包裹请求体，超限时 ParseMultipartForm 会返回错误。
	LimitBody(w, r, a.Cfg.MaxUploadBytes)
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		if isMaxBytesError(err) {
			writeErr(w, http.StatusRequestEntityTooLarge, "上传内容超过大小上限")
			return
		}
		writeErr(w, 400, "需要 multipart/form-data")
		return
	}
	id := genID()
	baseDir := filepath.Join(a.Cfg.DataDir, "task", id)
	uploadDir := filepath.Join(baseDir, "upload")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		writeErr(w, 500, err.Error())
		return
	}

	task := &db.Task{
		ID:                 id,
		Status:             "pending",
		Progress:           0,
		SourceType:         "file",
		SummaryPreference:  r.FormValue("summary_preference"),
		ScreenshotMode:     defaultStr(r.FormValue("screenshot_mode"), "simple"),
		ScreenshotInterval: atoiDef(r.FormValue("screenshot_interval"), a.Cfg.DefaultScreenshotInterval),
		ReadingDepth:       atoiDef(r.FormValue("reading_depth"), 2),
		SummaryProfileID:   atoiDefault(r.FormValue("summary_profile_id"), 0),
		ASRProfileID:       atoiDefault(r.FormValue("asr_profile_id"), 0),
		VisionProfileID:    atoiDefault(r.FormValue("vision_profile_id"), 0),
		ReadingProfileID:   atoiDefault(r.FormValue("reading_comprehension_profile_id"), 0),
		GenerateAIViaASR:   boolForm(r.FormValue("generate_ai_subtitle_via_asr")),
		GenerateVision:     boolForm(r.FormValue("generate_vision_subtitle")),
		CostSummary:        "{}",
	}

	videoURL := strings.TrimSpace(r.FormValue("video_url"))
	if videoURL != "" {
		task.SourceType = "url"
		task.VideoURL = videoURL
		task.Filename = filenameFromURL(videoURL)
	} else if f, hdr, err := r.FormFile("video_file"); err == nil {
		defer f.Close()
		dst := filepath.Join(uploadDir, "video")
		if err := saveUpload(f, dst); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		task.SourceType = "file"
		task.VideoPath = dst
		task.Filename = SanitizeFilename(hdr.Filename)
	} else {
		writeErr(w, 400, "需要 video_file 或 video_url")
		return
	}

	if err := a.DB.CreateTask(task); err != nil {
		writeErr(w, 500, err.Error())
		return
	}

	for _, kind := range []string{"native", "ai", "vision", "danmu"} {
		if f, hdr, err := r.FormFile("subtitle_" + kind); err == nil {
			name := SanitizeFilename(hdr.Filename)
			// 用 SafeJoin 确保落盘路径不越出 uploadDir。
			dst, jerr := SafeJoin(uploadDir, kind+"_"+name)
			if jerr == nil {
				if err := saveUpload(f, dst); err == nil {
					_ = a.DB.AddTaskSubtitle(&db.TaskSubtitle{TaskID: id, Kind: kind, OrigName: name, StoredPath: dst})
				}
			}
			f.Close()
		}
	}

	// knowledge bases: repeated "kb" = "id:version"; priority starts at 1 in selection order
	for i, kv := range r.MultipartForm.Value["kb"] {
		parts := strings.SplitN(kv, ":", 2)
		if len(parts) != 2 {
			continue
		}
		kbID, _ := strconv.ParseInt(parts[0], 10, 64)
		ver, _ := strconv.Atoi(parts[1])
		if kbID > 0 && ver > 0 {
			_ = a.DB.AddTaskKB(&db.TaskKB{KBID: kbID, Version: ver, Priority: i + 1}, id)
		}
	}

	a.Worker.Enqueue(id)
	writeJSON(w, 201, task)
}

func (a *API) listTasks(w http.ResponseWriter, r *http.Request) {
	ts, err := a.DB.ListTasks()
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, ts)
}

func (a *API) getTask(w http.ResponseWriter, r *http.Request) {
	t, err := a.DB.GetTask(r.PathValue("id"))
	if err != nil {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, t)
}

func (a *API) taskStages(w http.ResponseWriter, r *http.Request) {
	st, err := a.DB.ListStages(r.PathValue("id"))
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, st)
}

func (a *API) taskResult(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := a.DB.GetTask(id)
	if err != nil {
		writeErr(w, 404, "not found")
		return
	}
	outDir := filepath.Join(a.Cfg.DataDir, "task", id, "output")
	subs, _ := a.DB.ListTaskSubtitles(id)
	kbs, _ := a.DB.ListTaskKBs(id)
	stages, _ := a.DB.ListStages(id)
	var cost map[string]any
	_ = json.Unmarshal([]byte(t.CostSummary), &cost)

	writeJSON(w, 200, map[string]any{
		"task":             t,
		"cost":             cost,
		"subtitles":        subs,
		"knowledge_bases":  kbs,
		"stages":           stages,
		"summary_md":       readFileStr(filepath.Join(outDir, "summary.md")),
		"knowledge_md":     readFileStr(filepath.Join(outDir, "knowledge.md")),
	})
}

func (a *API) taskFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	name := filepath.Base(r.PathValue("name"))
	var path string
	switch name {
	case "summary.md", "knowledge.md":
		path = filepath.Join(a.Cfg.DataDir, "task", id, "output", name)
	case "fused.srt", "asr.srt", "vision.srt":
		path = filepath.Join(a.Cfg.DataDir, "task", id, "subs", name)
	default:
		writeErr(w, 404, "文件不允许下载")
		return
	}
	if _, err := os.Stat(path); err != nil {
		writeErr(w, 404, "not found")
		return
	}
	http.ServeFile(w, r, path)
}

func (a *API) retryTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := a.DB.GetTask(id); err != nil {
		writeErr(w, 404, "not found")
		return
	}
	// 断点续跑：保留已完成(done/skipped)阶段，仅重置失败/未完成阶段，重跑时跳过已完成阶段。
	kept, _ := a.DB.ResetNonDoneStages(id)
	if err := a.DB.UpdateTaskStatus(id, "pending", 0, ""); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	a.Worker.Enqueue(id)
	writeJSON(w, 200, map[string]any{"status": "queued", "resumed_from": kept})
}

// cleanupTaskCache deletes the task's temporary intermediates; only when paused/failed/done.
func (a *API) cleanupTaskCache(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := a.DB.GetTask(id)
	if err != nil {
		writeErr(w, 404, "not found")
		return
	}
	switch t.Status {
	case "done", "failed", "paused":
	default:
		writeErr(w, 400, "任务状态为 "+t.Status+"，仅 paused/failed/done 可清理缓存")
		return
	}
	root := filepath.Join(a.Cfg.TmpDir, "intermediates")
	dir := filepath.Join(root, filepath.Base(id))
	// 安全校验：待删目录必须严格位于 intermediates 之下，防止 id 造成路径穿越而误删。
	if !WithinDir(root, dir) || filepath.Clean(dir) == filepath.Clean(root) {
		writeErr(w, 400, "非法任务路径")
		return
	}
	if err := os.RemoveAll(dir); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok", "cleaned": dir})
}

func (a *API) deleteTask(w http.ResponseWriter, r *http.Request) {
	if err := a.DB.DeleteTask(r.PathValue("id")); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func saveUpload(src io.Reader, dst string) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, src)
	return err
}

func readFileStr(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return string(b)
}

func defaultStr(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func filenameFromURL(u string) string {
	base := u
	if i := strings.IndexAny(base, "?#"); i >= 0 {
		base = base[:i]
	}
	base = base[strings.LastIndex(base, "/")+1:]
	if base == "" {
		return "video"
	}
	return base
}
