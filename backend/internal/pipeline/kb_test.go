package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// TestIngestAndLoadCSV 校验上传 CSV 术语后可按版本加载，term_count 正确。
func TestIngestAndLoadCSV(t *testing.T) {
	dataDir := t.TempDir()
	content := []byte("standard,aliases\n斯涅尔定律,斯涅尔|snell\n折射率,refractive\n")
	vdir, sha, newBytes, err := IngestVersion(dataDir, 1, 1, "terms.csv", content)
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if sha == "" || newBytes != int64(len(content)) {
		t.Fatalf("sha/newBytes 异常: sha=%s newBytes=%d", sha, newBytes)
	}
	terms, err := LoadTermsFromVersion(vdir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(terms) != 2 {
		t.Fatalf("期望 2 个术语，实际 %d: %+v", len(terms), terms)
	}
	if terms[0].Standard != "斯涅尔定律" || len(terms[0].Aliases) != 2 {
		t.Fatalf("术语解析错误: %+v", terms[0])
	}
	if got := CountTermsInVersion(vdir); got != 2 {
		t.Fatalf("CountTerms 期望 2，实际 %d", got)
	}
}

// TestIngestJSON 校验 JSON 格式术语解析。
func TestIngestJSON(t *testing.T) {
	dataDir := t.TempDir()
	content := []byte("{\"terms\":[{\"standard\":\"全反射\",\"aliases\":[\"total internal reflection\"]},{\"standard\":\"临界角\",\"aliases\":[]}]}")
	vdir, _, _, err := IngestVersion(dataDir, 2, 1, "kb.json", content)
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	terms, err := LoadTermsFromVersion(vdir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(terms) != 2 || terms[0].Standard != "全反射" {
		t.Fatalf("JSON 术语解析错误: %+v", terms)
	}
}

// TestContentAddressingDedupe 校验相同内容只写一份 blob（newBytes 第二次为 0）。
func TestContentAddressingDedupe(t *testing.T) {
	dataDir := t.TempDir()
	content := []byte("standard,aliases\nA,B\n")
	_, sha1, nb1, err := IngestVersion(dataDir, 1, 1, "a.csv", content)
	if err != nil {
		t.Fatalf("ingest1: %v", err)
	}
	_, sha2, nb2, err := IngestVersion(dataDir, 1, 2, "a.csv", content)
	if err != nil {
		t.Fatalf("ingest2: %v", err)
	}
	if sha1 != sha2 {
		t.Fatalf("相同内容 sha 应相同: %s vs %s", sha1, sha2)
	}
	if nb1 == 0 || nb2 != 0 {
		t.Fatalf("重复内容不应重复计入 newBytes: nb1=%d nb2=%d", nb1, nb2)
	}
}

// TestDeleteVersionAndGC 校验删除版本后无引用的 blob 被回收，其它版本引用的 blob 保留。
func TestDeleteVersionAndGC(t *testing.T) {
	dataDir := t.TempDir()
	shared := []byte("standard,aliases\nShared,X\n")
	unique := []byte("standard,aliases\nUnique,Y\n")
	// v1: shared; v2: shared + unique（两次 IngestVersion 各写一个 manifest，但需同一 blob 目录）
	// 先写入 shared 到 v1
	v1dir, sharedSha, _, err := IngestVersion(dataDir, 1, 1, "s.csv", shared)
	if err != nil {
		t.Fatalf("ingest v1: %v", err)
	}
	// 手工构造 v2 引用 shared 与 unique，验证 GC 只回收 unique
	v2dir := filepath.Join(dataDir, "knowledge-bases", "1", "v2")
	if err := os.MkdirAll(v2dir, 0o755); err != nil {
		t.Fatalf("mkdir v2: %v", err)
	}
	uSum := sha256hex(unique)
	uBlob := blobPath(dataDir, uSum)
	if err := os.MkdirAll(filepath.Dir(uBlob), 0o755); err != nil {
		t.Fatalf("mkdir blob: %v", err)
	}
	if err := os.WriteFile(uBlob, unique, 0o644); err != nil {
		t.Fatalf("write blob: %v", err)
	}
	mf := `{"files":[{"name":"s.csv","sha256":"` + sharedSha + `","size":` + itoa(len(shared)) + `},{"name":"u.csv","sha256":"` + uSum + `","size":` + itoa(len(unique)) + `}]}`
	if err := os.WriteFile(filepath.Join(v2dir, "manifest.json"), []byte(mf), 0o644); err != nil {
		t.Fatalf("write v2 manifest: %v", err)
	}
	_ = v1dir

	// 删除 v1：shared 仍被 v2 引用 => 应保留
	if err := DeleteVersionAndGC(dataDir, v1dir); err != nil {
		t.Fatalf("gc v1: %v", err)
	}
	if _, err := os.Stat(blobPath(dataDir, sharedSha)); err != nil {
		t.Fatalf("共享 blob 应保留，却被删除: %v", err)
	}
	if _, err := os.Stat(v1dir); !os.IsNotExist(err) {
		t.Fatalf("v1 目录应被删除")
	}

	// 删除 v2：shared 与 unique 均无引用 => 应回收
	if err := DeleteVersionAndGC(dataDir, v2dir); err != nil {
		t.Fatalf("gc v2: %v", err)
	}
	if _, err := os.Stat(blobPath(dataDir, sharedSha)); !os.IsNotExist(err) {
		t.Fatalf("共享 blob 应被回收")
	}
	if _, err := os.Stat(uBlob); !os.IsNotExist(err) {
		t.Fatalf("unique blob 应被回收")
	}
}
func sha256hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
