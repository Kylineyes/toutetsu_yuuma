package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/config"
	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/db"
	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/worker"
)

// API wires HTTP handlers.
type API struct {
	DB     *db.DB
	Cfg    *config.Config
	Worker *worker.Worker
}

// Routes builds the HTTP handler.
func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", a.handleHealth)
	mux.HandleFunc("GET /api/v1/meta", a.handleMeta)
	mux.HandleFunc("POST /api/v1/login", a.login)
	mux.HandleFunc("POST /api/v1/logout", a.logout)
	mux.HandleFunc("GET /api/v1/session", a.session)

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
	mux.HandleFunc("DELETE /api/v1/knowledge-bases/{id}/versions/{v}", a.deleteKBVersion)

	mux.HandleFunc("POST /api/v1/tasks", a.createTask)
	mux.HandleFunc("GET /api/v1/tasks", a.listTasks)
	mux.HandleFunc("GET /api/v1/tasks/{id}", a.getTask)
	mux.HandleFunc("GET /api/v1/tasks/{id}/result", a.taskResult)
	mux.HandleFunc("GET /api/v1/tasks/{id}/stages", a.taskStages)
	mux.HandleFunc("GET /api/v1/tasks/{id}/files/{name}", a.taskFile)
	mux.HandleFunc("POST /api/v1/tasks/{id}/retry", a.retryTask)
	mux.HandleFunc("DELETE /api/v1/tasks/{id}/cache", a.cleanupTaskCache)
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", a.deleteTask)

	return a.withAuth(mux)
}

func exemptPath(p string) bool {
	return p == "/api/v1/health" || p == "/api/v1/login" || p == "/api/v1/session"
}

func (a *API) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") || exemptPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		// third-party API token
		if a.Cfg.APIToken != "" && a.tokenMatches(r, a.Cfg.APIToken) {
			next.ServeHTTP(w, r)
			return
		}
		// web session cookie
		if a.Cfg.Password != "" && a.validSession(r) {
			next.ServeHTTP(w, r)
			return
		}
		// fully open
		if a.Cfg.APIToken == "" && a.Cfg.Password == "" {
			next.ServeHTTP(w, r)
			return
		}
		writeErr(w, http.StatusUnauthorized, "未授权：请登录或提供 API_TOKEN")
	})
}

func (a *API) tokenMatches(r *http.Request, want string) bool {
	got := r.Header.Get("X-API-Token")
	if got == "" {
		got = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	if got == "" {
		got = r.URL.Query().Get("token")
	}
	return got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func (a *API) sessionValue() string {
	mac := hmac.New(sha256.New, []byte(a.Cfg.SessionSecret))
	mac.Write([]byte("ok"))
	return hex.EncodeToString(mac.Sum(nil))
}

func (a *API) validSession(r *http.Request) bool {
	c, err := r.Cookie("session")
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(c.Value), []byte(a.sessionValue())) == 1
}

// ---------------- helpers ----------------

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(normalizeNilSlice(v))
}

// normalizeNilSlice 将 nil 切片替换为同类型的空切片，确保 JSON 编码为 [] 而非 null。
// 仅处理"切片"类型（含带方法的命名切片类型）；map/结构体/指针等其它 nil 语义保持不变。
func normalizeNilSlice(v any) any {
	if v == nil {
		return v
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice {
		return v
	}
	if !rv.IsNil() {
		return v
	}
	// 用 reflect.New(sliceType).Elem() 得到该切片类型的零值（非 nil 空切片）。
	return reflect.MakeSlice(rv.Type(), 0, 0).Interface()
}
func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
func genID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix()%100000)
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

// ---------------- health / meta / auth ----------------

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (a *API) handleMeta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"default_screenshot_interval": a.Cfg.DefaultScreenshotInterval,
		"max_frames":                  a.Cfg.MaxFrames,
		"subtitle_formats":            []string{"srt", "ass", "vtt", "txt"},
		"capabilities":                []string{"summary", "asr", "vision", "reading"},
		"chat_protocols":              []string{"openai_chat", "openai_responses", "anthropic_messages", "gemini"},
		"asr_protocols":               []string{"deepseek_asr", "openai_audio"},
		"default_reading_depth":       2,
		"login_required":              a.Cfg.Password != "",
	})
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if a.Cfg.Password == "" {
		writeJSON(w, 200, map[string]string{"status": "ok", "note": "未设置密码，无需登录"})
		return
	}
	if body.Password != a.Cfg.Password {
		writeErr(w, 401, "密码错误")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    a.sessionValue(),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 3600,
	})
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (a *API) session(w http.ResponseWriter, r *http.Request) {
	authed := a.Cfg.Password == "" || a.validSession(r)
	writeJSON(w, 200, map[string]any{"authenticated": authed, "login_required": a.Cfg.Password != ""})
}

// ---------------- profiles ----------------

var chatProtocols = map[string]bool{"openai_chat": true, "openai_responses": true, "anthropic_messages": true, "gemini": true}
var asrProtocols = map[string]bool{"deepseek_asr": true, "openai_audio": true, "openai_whisper": true}

func normalizeProtocol(capability, protocol string) (string, bool) {
	if protocol == "" {
		if capability == "asr" {
			return "deepseek_asr", true
		}
		return "openai_chat", true
	}
	if capability == "asr" {
		return protocol, asrProtocols[protocol]
	}
	return protocol, chatProtocols[protocol]
}

func (a *API) listProfiles(w http.ResponseWriter, r *http.Request) {
	ps, err := a.DB.ListProfiles(r.URL.Query().Get("capability"))
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
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
	proto, ok := normalizeProtocol(p.Capability, p.Protocol)
	if !ok {
		writeErr(w, 400, "protocol 与 capability 不匹配")
		return
	}
	p.Protocol = proto
	if p.TimeoutSeconds <= 0 {
		p.TimeoutSeconds = 180
	}
	if p.Extra == "" {
		p.Extra = "{}"
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
	p, err := a.DB.GetProfile(atoiDefault(r.PathValue("id"), 0))
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
	ex, err := a.DB.GetProfile(id)
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
		p.Capability = ex.Capability
	}
	if p.Name == "" {
		p.Name = ex.Name
	}
	if p.BaseURL == "" {
		p.BaseURL = ex.BaseURL
	}
	if p.Model == "" {
		p.Model = ex.Model
	}
	if p.APIKey == "" || p.APIKey == "***" {
		p.APIKey = ex.APIKey
	}
	if p.TimeoutSeconds <= 0 {
		p.TimeoutSeconds = ex.TimeoutSeconds
	}
	if p.Extra == "" {
		p.Extra = ex.Extra
	}
	proto, ok := normalizeProtocol(p.Capability, p.Protocol)
	if !ok {
		writeErr(w, 400, "protocol 与 capability 不匹配")
		return
	}
	p.Protocol = proto
	if err := a.DB.UpdateProfile(&p); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (a *API) deleteProfile(w http.ResponseWriter, r *http.Request) {
	if err := a.DB.DeleteProfile(atoiDefault(r.PathValue("id"), 0)); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
