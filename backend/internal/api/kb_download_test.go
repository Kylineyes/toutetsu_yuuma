package api

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// newMultipartFile 构造一个带单文件字段 file 的 multipart 请求体。
func newMultipartFile(t *testing.T, name, content string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", name)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write([]byte(content)); err != nil {
		t.Fatalf("write file: %v", err)
	}
	mw.Close()
	return &buf, mw.FormDataContentType()
}

// TestDownloadKBVersion_UploadThenDownload 上传版本后下载应得到原始文件名与内容。
func TestDownloadKBVersion_UploadThenDownload(t *testing.T) {
	a := newTestAPI(t)
	kbID, err := a.DB.CreateKnowledgeBase("测试库")
	if err != nil {
		t.Fatalf("create kb: %v", err)
	}

	content := "term,alias\n茅台,Maotai\n"
	body, ct := newMultipartFile(t, "术语.csv", content)

	// 上传版本 v1
	upReq := httptest.NewRequest("POST", "/api/v1/knowledge-bases/"+strconv.FormatInt(kbID, 10)+"/versions", body)
	upReq.Header.Set("Content-Type", ct)
	upReq.SetPathValue("id", strconv.FormatInt(kbID, 10))
	upRec := httptest.NewRecorder()
	a.uploadKBVersion(upRec, upReq)
	if upRec.Code != 201 {
		t.Fatalf("上传期望 201，实际 %d：%s", upRec.Code, upRec.Body.String())
	}

	// 下载 v1
	dlReq := httptest.NewRequest("GET", "/api/v1/knowledge-bases/"+strconv.FormatInt(kbID, 10)+"/versions/1/file", nil)
	dlReq.SetPathValue("id", strconv.FormatInt(kbID, 10))
	dlReq.SetPathValue("v", "1")
	dlRec := httptest.NewRecorder()
	a.downloadKBVersion(dlRec, dlReq)

	if dlRec.Code != 200 {
		t.Fatalf("下载期望 200，实际 %d：%s", dlRec.Code, dlRec.Body.String())
	}
	if got := dlRec.Body.String(); got != content {
		t.Fatalf("下载内容不一致：期望 %q 实际 %q", content, got)
	}
	cd := dlRec.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") {
		t.Fatalf("Content-Disposition 应为 attachment，实际 %q", cd)
	}
}

// TestDownloadKBVersion_NotFound 不存在的版本应返回 404。
func TestDownloadKBVersion_NotFound(t *testing.T) {
	a := newTestAPI(t)
	kbID, err := a.DB.CreateKnowledgeBase("空库")
	if err != nil {
		t.Fatalf("create kb: %v", err)
	}
	req := httptest.NewRequest("GET", "/api/v1/knowledge-bases/"+strconv.FormatInt(kbID, 10)+"/versions/9/file", nil)
	req.SetPathValue("id", strconv.FormatInt(kbID, 10))
	req.SetPathValue("v", "9")
	rec := httptest.NewRecorder()
	a.downloadKBVersion(rec, req)
	if rec.Code != 404 {
		t.Fatalf("不存在版本期望 404，实际 %d：%s", rec.Code, rec.Body.String())
	}
}
