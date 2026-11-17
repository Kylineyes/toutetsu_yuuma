package pipeline

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/subtitle"
)

// kbFile is the JSON schema accepted for a knowledge base version file.
type kbFile struct {
	Terms []struct {
		Standard string   `json:"standard"`
		Aliases  []string `json:"aliases"`
		Note     string   `json:"note"`
	} `json:"terms"`
}

// LoadTerms reads all term files in a knowledge-base version directory.
func LoadTerms(dir string) ([]subtitle.Term, int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, err
	}
	var terms []subtitle.Term
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name())
		ext := strings.ToLower(filepath.Ext(e.Name()))
		switch ext {
		case ".json":
			data, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			var kf kbFile
			if json.Unmarshal(data, &kf) == nil && len(kf.Terms) > 0 {
				for _, t := range kf.Terms {
					if strings.TrimSpace(t.Standard) != "" {
						terms = append(terms, subtitle.Term{Standard: t.Standard, Aliases: t.Aliases})
					}
				}
			}
		case ".csv", ".txt", ".tsv":
			terms = append(terms, loadCSVTerms(p, ext == ".tsv")...)
		}
	}
	return terms, len(terms), nil
}

func loadCSVTerms(path string, tsv bool) []subtitle.Term {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	r := csv.NewReader(f)
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
			continue // header
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
