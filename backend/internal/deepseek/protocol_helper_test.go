package deepseek

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTempAudio 写一个最小 wav 头+数据到临时文件，返回路径（供 multipart 上传用）。
func writeTempAudio(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "a.wav")
	// 44 字节 RIFF 头 + 少许数据
	hdr := []byte{'R', 'I', 'F', 'F', 36, 0, 0, 0, 'W', 'A', 'V', 'E', 'f', 'm', 't', ' ', 16, 0, 0, 0, 1, 0, 1, 0, 64, 31, 0, 0, 128, 62, 0, 0, 2, 0, 16, 0, 'd', 'a', 't', 'a', 0, 0, 0, 0}
	if err := os.WriteFile(p, hdr, 0o644); err != nil {
		t.Fatalf("write audio: %v", err)
	}
	return p
}
