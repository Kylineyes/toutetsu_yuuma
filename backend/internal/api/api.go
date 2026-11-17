package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/config"
	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/db"
	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/pipeline"
	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/worker"
)

// API wires HTTP handlers.
type API struct {
	DB     *db.DB
	Cfg    *config.Config
	Worker *worker.Worker
}

// Routes builds the HTTP handler with optional API-token auth.
func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", a.handleHealth)
	mux.HandleFunc("GET /api/v1/meta", a.handleMeta)
	mux.HandleFunc("GET /api/v1/settings", a.getSettings)
	mux.HandleFunc("PUT /api/v1/settings", a.putSettings)

	mux.HandleFunc("GET /api/v1/profiles", a.listProfiles)
	mux.HandleFunc("POST /api/v1/profiles", a.createProfile)
	mux.HandleFunc("GET /api/v1/profiles/{id}", a.getProfile)
	mux.HandleFunc("PUT /api/v1/profiles/{id}", a.updateProfile)
	mux.HandleFunc("DELETE /api/v1/profiles/{id}", a.deleteProfile)

	mux.HandleFunc("GET /api/v1/knowledge-bases", a.listKBs)
	mux.HandleFunc("POST /api/v1/knowledge-bases", a.createKB)
	mux.HandleFunc("GET /api/v1/knowledge-bases/{id}", a.getKB)
	mux.HandleFunc("DELETE /api/v1/knowledge-bases/{id}", a.deleteKB)
	mux.HandleFunc("GET /api/v1/knowledge-bases/{id}/versions", a.listKBVersions)
	mux.HandleFunc("POST /api/v1/knowledge-bases/{id}/versions", a.uploadKBVersion)
	mux.HandleFunc("GET /api/v1/knowledge-bases/{id}/versions/{v}", a.getKBVersion)

	mux.HandleFunc("POST /api/v1/tasks", a.createTask)
	mux.HandleFunc("GET /api/v1/tasks", a.listTasks)
	mux.HandleFunc("GET /api/v1/tasks/{id}", a.getTask)
	mux.HandleFunc("GET /api/v1/tasks/{id}/result", a.taskResult)
	mux.HandleFunc("GET /api/v1/tasks/{id}/files/{name}", a.taskFile)
	mux.HandleFunc("POST /api/v1/tasks/{id}/retry", a.retryTask)
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", a.deleteTask)

	return a.withAuth(mux)
}

func (a *API) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") && !exemptPath(r.URL.Path) {
			// api_token 明文存放在 DB 的 settings 表
			token, _ := a.DB.GetSetting("api_token")
			if token != "" {
				tok := r.Header.Get("X-API-Token")
				if tok == "" {
					tok = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
				}
				if tok == "" {
					tok = r.URL.Query().Get("token")
				}
				if tok != token {
					writeErr(w, http.StatusUnauthorized, "invalid api token")
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// exemptPath: health 与 settings 永远免鉴权（便于初次设置令牌）。
func exemptPath(p string) bool {
	return p == "/api/v1/health" || strings.HasPrefix(p, "/api/v1/settings")
}

// helpers
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
func genID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%d-%s", time.Now().Unix(), hex.EncodeToString(b))
}
func atoiDefault(s string, def int64) int64 {
	if s == "" {
		return def
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	return def
}
func atoiDef(s string, def int) int {
	if s == "" {
		return def
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}
func boolForm(s string) bool {
	return s == "1" || strings.EqualFold(s, "true") || strings.EqualFold(s, "on")
}

// ---------------- health / meta ----------------

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (a *API) handleMeta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"default_screenshot_interval": a.Cfg.DefaultScreenshotInterval,
		"max_frames":                  a.Cfg.MaxFrames,
		"subtitle_formats":            []string{"srt", "ass", "vtt", "txt"},
		"capabilities":                []string{"summary", "asr", "vision", "reading"},
		"default_reading_depth":       2,
	})
}

// ---------------- settings ----------------

// getSettings returns the stored api_token (plaintext, by design).
func (a *API) getSettings(w http.ResponseWriter, r *http.Request) {
	token, _ := a.DB.GetSetting("api_token")
	writeJSON(w, 200, map[string]string{"api_token": token})
}

// putSettings upserts settings; currently only api_token. Empty string clears it.
func (a *API) putSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		APIToken string `json:"api_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	if err := a.DB.SetSetting("api_token", strings.TrimSpace(body.APIToken)); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

// ---------------- profiles ----------------

func (a *API) listProfiles(w http.ResponseWriter, r *http.Request) {
	caps := r.URL.Query().Get("capability")
	ps, err := a.DB.ListProfiles(caps)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	// never leak api_key in list responses
	for i := range ps {
		if ps[i].APIKey != "" {
			ps[i].APIKey = "***"
		}
	}
	writeJSON(w, 200, ps)
}

func (a *API) createProfile(w http.ResponseWriter, r *http.Request) {
	var p db.Profile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	if p.Capability == "" || p.Name == "" || p.BaseURL == "" || p.Model == "" {
		writeErr(w, 400, "capability/name/base_url/model 必填")
		return
	}
	if p.BaseURL == "" {
		p.BaseURL = "https://api.deepseek.com"
	}
	id, err := a.DB.CreateProfile(&p)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	p.ID = id
	if p.APIKey != "" {
		p.APIKey = "***"
	}
	writeJSON(w, 201, p)
}

func (a *API) getProfile(w http.ResponseWriter, r *http.Request) {
	id := atoiDefault(r.PathValue("id"), 0)
	p, err := a.DB.GetProfile(id)
	if err != nil {
		writeErr(w, 404, "not found")
		return
	}
	if p.APIKey != "" {
		p.APIKey = "***"
	}
	writeJSON(w, 200, p)
}

func (a *API) updateProfile(w http.ResponseWriter, r *http.Request) {
	id := atoiDefault(r.PathValue("id"), 0)
	existing, err := a.DB.GetProfile(id)
	if err != nil {
		writeErr(w, 404, "not found")
		return
	}
	var p db.Profile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	p.ID = id
	if p.Capability == "" {
		p.Capability = existing.Capability
	}
	if p.Name == "" {
		p.Name = existing.Name
	}
	if p.BaseURL == "" {
		p.BaseURL = existing.BaseURL
	}
	if p.Model == "" {
		p.Model = existing.Model
	}
	// empty or masked api_key keeps the old one
	if p.APIKey == "" || p.APIKey == "***" {
		p.APIKey = existing.APIKey
	}
	if err := a.DB.UpdateProfile(&p); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (a *API) deleteProfile(w http.ResponseWriter, r *http.Request) {
	id := atoiDefault(r.PathValue("id"), 0)
	if err := a.DB.DeleteProfile(id); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

// ---------------- knowledge bases ----------------

func (a *API) listKBs(w http.ResponseWriter, r *http.Request) {
	kbs, err := a.DB.ListKnowledgeBases()
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, kbs)
}

func (a *API) createKB(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		writeErr(w, 400, "name 必填")
		return
	}
	id, err := a.DB.CreateKnowledgeBase(strings.TrimSpace(body.Name))
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, map[string]any{"id": id, "name": body.Name, "latest_version": 0})
}

func (a *API) getKB(w http.ResponseWriter, r *http.Request) {
	id := atoiDefault(r.PathValue("id"), 0)
	k, err := a.DB.GetKnowledgeBase(id)
	if err != nil {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, k)
}

func (a *API) deleteKB(w http.ResponseWriter, r *http.Request) {
	id := atoiDefault(r.PathValue("id"), 0)
	if err := a.DB.DeleteKnowledgeBase(id); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (a *API) listKBVersions(w http.ResponseWriter, r *http.Request) {
	id := atoiDefault(r.PathValue("id"), 0)
	vs, err := a.DB.ListKBVersions(id)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, vs)
}

func (a *API) uploadKBVersion(w http.ResponseWriter, r *http.Request) {
	id := atoiDefault(r.PathValue("id"), 0)
	kb, err := a.DB.GetKnowledgeBase(id)
	if err != nil {
		writeErr(w, 404, "knowledge base not found")
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeErr(w, 400, "需要 multipart/form-data")
		return
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		writeErr(w, 400, "缺少 file 字段")
		return
	}
	defer f.Close()

	ver := kb.LatestVersion + 1
	dir := filepath.Join(a.Cfg.DataDir, "knowledge-bases", strconv.FormatInt(id, 10), fmt.Sprintf("v%d", ver))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	name := filepath.Base(hdr.Filename)
	dst := filepath.Join(dir, name)
	out, err := os.Create(dst)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(out, h), f); err != nil {
		out.Close()
		writeErr(w, 500, err.Error())
		return
	}
	out.Close()

	_, termCount, _ := pipeline.LoadTerms(dir)
	v := &db.KBVersion{KBID: id, DirPath: dir, FileName: name, TermCount: termCount, SHA256: hex.EncodeToString(h.Sum(nil))}
	saved, err := a.DB.AddKBVersion(v)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, saved)
}

func (a *API) getKBVersion(w http.ResponseWriter, r *http.Request) {
	id := atoiDefault(r.PathValue("id"), 0)
	v, _ := strconv.Atoi(r.PathValue("v"))
	kv, err := a.DB.GetKBVersion(id, v)
	if err != nil {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, kv)
}
