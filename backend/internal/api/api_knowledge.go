package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/db"
	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/pipeline"
)

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
	k, err := a.DB.GetKnowledgeBase(atoiDefault(r.PathValue("id"), 0))
	if err != nil {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, k)
}

func (a *API) deleteKB(w http.ResponseWriter, r *http.Request) {
	if err := a.DB.DeleteKnowledgeBase(atoiDefault(r.PathValue("id"), 0)); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (a *API) listKBVersions(w http.ResponseWriter, r *http.Request) {
	vs, err := a.DB.ListKBVersions(atoiDefault(r.PathValue("id"), 0))
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, vs)
}

// uploadKBVersion ingests one term file into content-addressed storage and records a new version.
func (a *API) uploadKBVersion(w http.ResponseWriter, r *http.Request) {
	id := atoiDefault(r.PathValue("id"), 0)
	kb, err := a.DB.GetKnowledgeBase(id)
	if err != nil {
		writeErr(w, 404, "knowledge base not found")
		return
	}
	LimitBody(w, r, a.Cfg.MaxUploadBytes)
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		if isMaxBytesError(err) {
			writeErr(w, http.StatusRequestEntityTooLarge, "上传内容超过大小上限")
			return
		}
		writeErr(w, 400, "需要 multipart/form-data")
		return
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		writeErr(w, 400, "缺少 file 字段")
		return
	}
	defer f.Close()
	content, err := io.ReadAll(f)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	ver := kb.LatestVersion + 1
	safeName := SanitizeFilename(hdr.Filename)
	dirPath, sha, newBytes, err := pipeline.IngestVersion(a.Cfg.DataDir, id, ver, safeName, content)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	termCount := pipeline.CountTermsInVersion(dirPath)
	v := &db.KBVersion{KBID: id, DirPath: dirPath, FileName: safeName, SHA256: sha, TermCount: termCount, SizeBytes: newBytes}
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

// deleteKBVersion deletes a version if not referenced by tasks and not the latest; GC unreferenced blobs.
func (a *API) deleteKBVersion(w http.ResponseWriter, r *http.Request) {
	id := atoiDefault(r.PathValue("id"), 0)
	v, _ := strconv.Atoi(r.PathValue("v"))
	kb, err := a.DB.GetKnowledgeBase(id)
	if err != nil {
		writeErr(w, 404, "knowledge base not found")
		return
	}
	if v == kb.LatestVersion {
		writeErr(w, 400, "不能删除最新版本")
		return
	}
	kv, err := a.DB.GetKBVersion(id, v)
	if err != nil {
		writeErr(w, 404, "版本不存在")
		return
	}
	refs, _ := a.DB.CountVersionRefs(id, v)
	if refs > 0 {
		writeErr(w, 400, "该版本被 "+strconv.Itoa(refs)+" 个任务引用，不可删除")
		return
	}
	if err := pipeline.DeleteVersionAndGC(a.Cfg.DataDir, kv.DirPath); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if err := a.DB.DeleteKBVersion(id, v); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
