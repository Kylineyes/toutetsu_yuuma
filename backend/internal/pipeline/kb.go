package pipeline

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/subtitle"
)

// manifest describes one knowledge-base version (content-addressed references).
type manifest struct {
	Files []manifestFile `json:"files"`
}

type manifestFile struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

func blobsRoot(dataDir string) string {
	return filepath.Join(dataDir, "knowledge-bases", "blobs")
}

func blobPath(dataDir, sha string) string {
	prefix := "00"
	if len(sha) >= 2 {
		prefix = sha[:2]
	}
	return filepath.Join(blobsRoot(dataDir), prefix, sha)
}

// IngestVersion writes content-addressed blob(s) for one uploaded file and a version manifest.
// Returns the version dir, main file sha256, and newly-added bytes (for size_bytes).
func IngestVersion(dataDir string, kbID int64, version int, fileName string, content []byte) (string, string, int64, error) {
	sum := sha256.Sum256(content)
	sha := hex.EncodeToString(sum[:])
	bp := blobPath(dataDir, sha)
	var newBytes int64
	if _, err := os.Stat(bp); err != nil {
		if err := os.MkdirAll(filepath.Dir(bp), 0o755); err != nil {
			return "", "", 0, err
		}
		if err := os.WriteFile(bp, content, 0o644); err != nil {
			return "", "", 0, err
		}
		newBytes = int64(len(content))
	}
	versionDir := filepath.Join(dataDir, "knowledge-bases", itoa64(kbID), "v"+itoa(version))
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		return "", "", 0, err
	}
	mf := manifest{Files: []manifestFile{{Name: fileName, SHA256: sha, Size: int64(len(content))}}}
	raw, _ := json.MarshalIndent(mf, "", "  ")
	if err := os.WriteFile(filepath.Join(versionDir, "manifest.json"), raw, 0o644); err != nil {
		return "", "", 0, err
	}
	return versionDir, sha, newBytes, nil
}

// LoadTermsFromVersion parses terms from a version's manifest-referenced blobs.
func LoadTermsFromVersion(versionDir string) ([]subtitle.Term, error) {
	dataDir := dataDirFromVersionDir(versionDir)
	raw, err := os.ReadFile(filepath.Join(versionDir, "manifest.json"))
	if err != nil {
		return nil, err
	}
	var mf manifest
	if err := json.Unmarshal(raw, &mf); err != nil {
		return nil, err
	}
	var terms []subtitle.Term
	for _, f := range mf.Files {
		bp := blobPath(dataDir, f.SHA256)
		data, err := os.ReadFile(bp)
		if err != nil {
			continue
		}
		terms = append(terms, parseTermBytes(f.Name, data)...)
	}
	return terms, nil
}

// CountTermsInVersion returns the number of terms in a version.
func CountTermsInVersion(versionDir string) int {
	t, _ := LoadTermsFromVersion(versionDir)
	return len(t)
}

// DeleteVersionAndGC removes a version dir and any blobs no longer referenced by other manifests.
func DeleteVersionAndGC(dataDir string, versionDir string) error {
	raw, err := os.ReadFile(filepath.Join(versionDir, "manifest.json"))
	var mf manifest
	if err == nil {
		_ = json.Unmarshal(raw, &mf)
	}
	_ = os.RemoveAll(versionDir)
	// for each blob, check remaining references across all manifests
	for _, f := range mf.Files {
		if !blobReferenced(dataDir, f.SHA256) {
			_ = os.Remove(blobPath(dataDir, f.SHA256))
		}
	}
	return nil
}

func blobReferenced(dataDir, sha string) bool {
	found := false
	root := filepath.Join(dataDir, "knowledge-bases")
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() != "manifest.json" {
			return nil
		}
		raw, e := os.ReadFile(p)
		if e != nil {
			return nil
		}
		var mf manifest
		if json.Unmarshal(raw, &mf) != nil {
			return nil
		}
		for _, f := range mf.Files {
			if f.SHA256 == sha {
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	return found
}

func dataDirFromVersionDir(versionDir string) string {
	// versionDir = <dataDir>/knowledge-bases/{kbId}/v{n}
	return filepath.Dir(filepath.Dir(filepath.Dir(versionDir)))
}

// parseTermBytes parses term definitions from file bytes by extension.
func parseTermBytes(name string, data []byte) []subtitle.Term {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".json":
		var kf struct {
			Terms []struct {
				Standard string   `json:"standard"`
				Aliases  []string `json:"aliases"`
				Note     string   `json:"note"`
			} `json:"terms"`
		}
		if json.Unmarshal(data, &kf) == nil && len(kf.Terms) > 0 {
			var out []subtitle.Term
			for _, t := range kf.Terms {
				if strings.TrimSpace(t.Standard) != "" {
					out = append(out, subtitle.Term{Standard: t.Standard, Aliases: t.Aliases})
				}
			}
			return out
		}
		return nil
	case ".csv", ".txt", ".tsv":
		return parseCSVBytes(data)
	}
	return nil
}

func parseCSVBytes(data []byte) []subtitle.Term {
	r := csv.NewReader(strings.NewReader(string(data)))
	r.FieldsPerRecord = -1
	lines, err := r.ReadAll()
	if err != nil {
		return nil
	}
	var out []subtitle.Term
	for i, row := range lines {
		if len(row) == 0 {
			continue
		}
		standard := strings.TrimSpace(row[0])
		if i == 0 && strings.EqualFold(standard, "standard") {
			continue
		}
		if standard == "" {
			continue
		}
		var aliases []string
		if len(row) > 1 {
			aliases = splitAliases(row[1])
		}
		out = append(out, subtitle.Term{Standard: standard, Aliases: aliases})
	}
	return out
}

func splitAliases(s string) []string {
	f := func(r rune) bool { return r == '|' || r == ';' || r == '、' || r == '/' }
	var out []string
	for _, a := range strings.FieldsFunc(s, f) {
		a = strings.TrimSpace(a)
		if a != "" {
			out = append(out, a)
		}
	}
	return out
}

func itoa(i int) string     { return strconv.Itoa(i) }
func itoa64(i int64) string { return strconv.FormatInt(i, 10) }
