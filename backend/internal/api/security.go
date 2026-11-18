package api

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ErrPathTraversal 表示目标路径越出了允许的根目录。
var ErrPathTraversal = errors.New("非法路径：越出允许目录")

// ErrUploadTooLarge 表示上传超过大小上限。
var ErrUploadTooLarge = errors.New("上传内容超过大小上限")

// SafeJoin 将 name 安全地拼接在 base 目录之下，并确保结果不会越出 base。
// 任何包含 ".." 或绝对路径、解析后 parent 不等于 base 的输入都会被拒绝。
func SafeJoin(base, name string) (string, error) {
	cleanBase := filepath.Clean(base)
	// 取文件名部分，剥离任何目录成分（防 ../../etc/passwd 之类）。
	safeName := filepath.Base(filepath.Clean(strings.ReplaceAll(name, "\\", "/")))
	if safeName == "." || safeName == ".." || safeName == string(filepath.Separator) || safeName == "" {
		return "", ErrPathTraversal
	}
	dst := filepath.Join(cleanBase, safeName)
	// 二次校验：解析后必须仍在 base 之内。
	rel, err := filepath.Rel(cleanBase, dst)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", ErrPathTraversal
	}
	return dst, nil
}

// WithinDir 校验 path 是否位于 dir 之下（含 dir 本身），用于清理等操作前的路径约束。
func WithinDir(dir, path string) bool {
	cleanDir := filepath.Clean(dir)
	cleanPath := filepath.Clean(path)
	rel, err := filepath.Rel(cleanDir, cleanPath)
	if err != nil || filepath.IsAbs(rel) {
		return false
	}
	return rel == "." || !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// LimitBody 用 MaxBytesReader 包裹请求体，超过 limit 时返回 413 并标记。
// 供 multipart 上传入口在 ParseMultipartForm 之前调用。
func LimitBody(w http.ResponseWriter, r *http.Request, limit int64) {
	if limit > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, limit)
	}
}

// CopyLimited 将 src 拷入 dst 文件，最多写 limit 字节；超出则删掉半成品并返回 ErrUploadTooLarge。
func CopyLimited(src io.Reader, dst string, limit int64) (int64, error) {
	f, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	var n int64
	var cpErr error
	if limit > 0 {
		// 多读 1 字节以判断是否超限。
		n, cpErr = io.Copy(f, io.LimitReader(src, limit+1))
	} else {
		n, cpErr = io.Copy(f, src)
	}
	if cerr := f.Close(); cerr != nil && cpErr == nil {
		cpErr = cerr
	}
	if cpErr != nil {
		_ = os.Remove(dst)
		return n, cpErr
	}
	if limit > 0 && n > limit {
		_ = os.Remove(dst)
		return n, ErrUploadTooLarge
	}
	return n, nil
}

// SanitizeFilename 返回仅含安全字符的文件名，用于展示/存储文件名与存储路径分段。
func SanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "file"
	}
	// 去除控制字符与路径分隔符，避免落盘时被曲解。
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == '/' || r == '\\' {
			return -1
		}
		return r
	}, name)
	if name == "" {
		return "file"
	}
	return name
}
// isMaxBytesError 判断错误是否来自 http.MaxBytesReader 的超限错误。
func isMaxBytesError(err error) bool {
	if err == nil {
		return false
	}
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) {
		return true
	}
	// multipart 解析在超限时可能返回含 "request body too large" 的错误。
	return strings.Contains(strings.ToLower(err.Error()), "request body too large")
}
