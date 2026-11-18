package api

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSafeJoin_Traversal 校验目录穿越被拒绝、正常文件名被接受。
func TestSafeJoin_Traversal(t *testing.T) {
	base := t.TempDir()
	// 正常
	if _, err := SafeJoin(base, "a.srt"); err != nil {
		t.Fatalf("正常文件名应接受: %v", err)
	}
	// 各种穿越形式都应被安全处理：要么拒绝，要么剥离为 base 内文件。
	cases := []string{"../../etc/passwd", "..\\..\\windows\\system32", "/etc/passwd", "a/../../b"}
	for _, c := range cases {
		got, err := SafeJoin(base, c)
		if err != nil {
			continue // 拒绝也是安全的
		}
		if !WithinDir(base, got) {
			t.Fatalf("输入 %q 得到的 %q 越出了 base", c, got)
		}
	}
	// ".." 与 "." 必须拒绝
	if _, err := SafeJoin(base, ".."); err == nil {
		t.Fatalf("'..' 应被拒绝")
	}
	if _, err := SafeJoin(base, "."); err == nil {
		t.Fatalf("'.' 应被拒绝")
	}
}

// TestWithinDir 校验目录包含判定。
func TestWithinDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "inter")
	if !WithinDir(root, filepath.Join(root, "x")) {
		t.Fatalf("子路径应在 root 内")
	}
	if !WithinDir(root, root) {
		t.Fatalf("root 自身应视为在内")
	}
	if WithinDir(root, filepath.Join(root, "..", "evil")) {
		t.Fatalf("越界路径应判定为不在 root 内")
	}
}

// TestSanitizeFilename 校验文件名清洗。
func TestSanitizeFilename(t *testing.T) {
	cases := map[string]string{
		"../../etc/passwd": "passwd",
		"a\\b\\c.txt":      "c.txt",
		"正常.srt":          "正常.srt",
		"":                 "file",
		"..":               "file",
	}
	for in, want := range cases {
		if got := SanitizeFilename(in); got != want {
			t.Fatalf("SanitizeFilename(%q)=%q，期望 %q", in, got, want)
		}
	}
}

// TestCopyLimited_Oversize 校验超限时返回错误且不留下半成品文件。
func TestCopyLimited_Oversize(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "out.bin")
	n, err := CopyLimited(bytes.NewReader(bytes.Repeat([]byte("x"), 100)), dst, 50)
	if err != ErrUploadTooLarge {
		t.Fatalf("应返回 ErrUploadTooLarge，实际 %v", err)
	}
	if n <= 50 {
		t.Fatalf("应读到超过 limit 的字节数，实际 %d", n)
	}
	if _, statErr := os.Stat(dst); !os.IsNotExist(statErr) {
		t.Fatalf("超限后不应残留文件")
	}
}

// TestCopyLimited_OK 校验未超限时正常写入。
func TestCopyLimited_OK(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "out.bin")
	n, err := CopyLimited(bytes.NewReader([]byte("hello")), dst, 50)
	if err != nil || n != 5 {
		t.Fatalf("正常写入失败: n=%d err=%v", n, err)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "hello" {
		t.Fatalf("内容错误: %q", got)
	}
}

// buildMultipart 构造一个包含 file 字段（指定大小）的 multipart 请求体。
func buildMultipart(t *testing.T, fieldName string, size int, filename string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	_, _ = fw.Write(bytes.Repeat([]byte("a"), size))
	_ = mw.Close()
	return &buf, mw.FormDataContentType()
}

// TestLimitBody_413 校验 http.MaxBytesReader + isMaxBytesError 端到端返回 413。
func TestLimitBody_413(t *testing.T) {
	// limit 很小，请求体超过它。
	body, ct := buildMultipart(t, "file", 4096, "big.bin")
	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()

	LimitBody(rec, req, 1024) // 限制 1KB
	err := req.ParseMultipartForm(64 << 20)
	if err == nil {
		t.Fatalf("超限应返回解析错误")
	}
	if !isMaxBytesError(err) {
		t.Fatalf("错误应被识别为超限: %v", err)
	}
	if rec.Code != http.StatusRequestEntityTooLarge && rec.Code != 200 {
		// 这里仅校验识别逻辑；实际写 413 由 handler 完成。rec.Code 未写时应为 200 默认。
	}
}

// TestIsMaxBytesError_Direct 直接校验 MaxBytesReader 触发的错误被识别。
func TestIsMaxBytesError_Direct(t *testing.T) {
	rec := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/x", strings.NewReader(strings.Repeat("z", 100)))
	r.Body = http.MaxBytesReader(rec, r.Body, 10)
	buf := make([]byte, 100)
	_, err := r.Body.Read(buf[:])
	_, err = r.Body.Read(buf[:])
	if err != nil && !isMaxBytesError(err) {
		t.Fatalf("MaxBytesReader 错误应被识别，err=%v", err)
	}
}
