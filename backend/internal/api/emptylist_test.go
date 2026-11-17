package api

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/config"
	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/db"
)

// newTestAPI 构造一个使用临时空数据库的 API（空库即触发 nil slice 场景）。
func newTestAPI(t *testing.T) *API {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := d.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return &API{DB: d, Cfg: &config.Config{DataDir: t.TempDir(), TmpDir: t.TempDir()}}
}

// assertJSONEmptyArray 断言响应体为 JSON 空数组 []（而非 null）。
func assertJSONEmptyArray(t *testing.T, body string) {
	t.Helper()
	trimmed := strings.TrimSpace(body)
	if trimmed == "null" {
		t.Fatalf("空列表不应编码为 null，实际: %q", trimmed)
	}
	if !strings.HasPrefix(trimmed, "[") {
		t.Fatalf("空列表应为 JSON 数组，实际: %q", trimmed)
	}
	// 反序列化到 []any 应为非 nil 空切片
	var arr []any
	if err := json.Unmarshal([]byte(trimmed), &arr); err != nil {
		t.Fatalf("反序列化失败: %v (body=%q)", err, trimmed)
	}
	if arr == nil {
		t.Fatalf("反序列化后不应为 nil 切片（说明是 null），body=%q", trimmed)
	}
	if len(arr) != 0 {
		t.Fatalf("空库应为空数组，实际长度 %d", len(arr))
	}
}

// TestListKBs_EmptyIsArray 空库时 knowledge-bases 列表应返回 []。
func TestListKBs_EmptyIsArray(t *testing.T) {
	a := newTestAPI(t)
	req := httptest.NewRequest("GET", "/api/v1/knowledge-bases", nil)
	rec := httptest.NewRecorder()
	a.listKBs(rec, req)
	if rec.Code != 200 {
		t.Fatalf("状态码期望 200，实际 %d", rec.Code)
	}
	assertJSONEmptyArray(t, rec.Body.String())
}

// TestListProfiles_EmptyIsArray 空库时 profiles 列表应返回 []。
func TestListProfiles_EmptyIsArray(t *testing.T) {
	a := newTestAPI(t)
	req := httptest.NewRequest("GET", "/api/v1/profiles", nil)
	rec := httptest.NewRecorder()
	a.listProfiles(rec, req)
	if rec.Code != 200 {
		t.Fatalf("状态码期望 200，实际 %d", rec.Code)
	}
	assertJSONEmptyArray(t, rec.Body.String())
}

// TestListTasks_EmptyIsArray 空库时 tasks 列表也应返回 []（通用兜底）。
func TestListTasks_EmptyIsArray(t *testing.T) {
	a := newTestAPI(t)
	req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	rec := httptest.NewRecorder()
	a.listTasks(rec, req)
	if rec.Code != 200 {
		t.Fatalf("状态码期望 200，实际 %d", rec.Code)
	}
	assertJSONEmptyArray(t, rec.Body.String())
}

// TestNormalizeNilSlice 校验 writeJSON 层：nil 切片→[]，非 nil 与其它类型语义不变。
func TestNormalizeNilSlice(t *testing.T) {
	// nil 切片
	var nilSlice []int
	out := normalizeNilSlice(nilSlice)
	if out == nil {
		t.Fatalf("nil 切片应规范为非 nil 空切片")
	}
	if s, ok := out.([]int); !ok || s == nil || len(s) != 0 {
		t.Fatalf("规范化结果应为空 []int，实际 %#v", out)
	}
	// 非 nil 切片原样
	orig := []string{"a"}
	if got := normalizeNilSlice(orig); got.([]string)[0] != "a" {
		t.Fatalf("非 nil 切片应原样返回")
	}
	// 非切片类型不受影响
	if got := normalizeNilSlice(nil); got != nil {
		t.Fatalf("nil 接口应原样返回 nil")
	}
	var m map[string]string
	if got := normalizeNilSlice(m); got != nil {
		// map nil 也应原样（保持 null 语义；不改成 {}）
		if _, ok := got.(map[string]string); !ok {
			t.Fatalf("nil map 应原样返回")
		}
	}
	// 非切片普通值
	if got := normalizeNilSlice("x"); got != "x" {
		t.Fatalf("普通值应原样返回")
	}
}

// TestWriteJSON_NilSliceIsArray 端到端：writeJSON 编码 nil []db.KnowledgeBase 得到 []。
func TestWriteJSON_NilSliceIsArray(t *testing.T) {
	rec := httptest.NewRecorder()
	var kbs []db.KnowledgeBase // nil
	writeJSON(rec, 200, kbs)
	assertJSONEmptyArray(t, rec.Body.String())
}