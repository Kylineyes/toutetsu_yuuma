package subtitle

import (
	"sort"
	"strings"
)

// Source is one subtitle source of cues.
type Source struct {
	Kind string // native | ai | asr | vision
	Cues []Cue
}

// Fuse merges sources into a single unified timeline.
// Priority: native > ai > asr. ASR fills gaps; vision adds on-screen text.
func Fuse(sources []Source) []Cue {
	byKind := map[string][]Cue{}
	for _, s := range sources {
		if len(s.Cues) > 0 {
			byKind[s.Kind] = append(byKind[s.Kind], s.Cues...)
		}
	}
	base := byKind["native"]
	if len(base) == 0 {
		base = byKind["ai"]
	}
	if len(base) == 0 {
		base = byKind["asr"]
	}
	base = cloneCues(base)

	// gap-fill with ASR only when base came from precise sources (native/ai)
	if len(byKind["asr"]) > 0 && len(base) > 0 && !sameCues(base, byKind["asr"]) {
		for _, a := range byKind["asr"] {
			if coverageRatio(base, a) < 0.3 {
				base = append(base, a)
			}
		}
	}

	// vision cues: append on-screen text lines
	for _, v := range byKind["vision"] {
		if strings.TrimSpace(v.Text) == "" {
			continue
		}
		base = append(base, v)
	}

	sort.SliceStable(base, func(i, j int) bool { return base[i].Start < base[j].Start })
	return dedupe(base)
}

func cloneCues(in []Cue) []Cue {
	out := make([]Cue, len(in))
	copy(out, in)
	return out
}

func sameCues(a, b []Cue) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	return a[0].Text == b[0].Text && a[0].Start == b[0].Start
}

func coverageRatio(base []Cue, c Cue) float64 {
	dur := c.End - c.Start
	if dur <= 0 {
		for _, b := range base {
			if c.Start >= b.Start && c.Start <= b.End {
				return 1
			}
		}
		return 0
	}
	var covered float64
	for _, b := range base {
		s := maxf(c.Start, b.Start)
		e := minf(c.End, b.End)
		if e > s {
			covered += e - s
		}
	}
	if covered > dur {
		covered = dur
	}
	return covered / dur
}

func dedupe(cues []Cue) []Cue {
	seen := map[string]bool{}
	var out []Cue
	for _, c := range cues {
		key := c.Text
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, c)
	}
	return out
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Term is one knowledge-base glossary entry.
type Term struct {
	Standard string
	Aliases  []string
}

// ApplyTerms corrects domain terms in every cue. Original cues are not mutated.
func ApplyTerms(cues []Cue, terms []Term) []Cue {
	if len(terms) == 0 {
		return cues
	}
	out := make([]Cue, len(cues))
	for i, c := range cues {
		t := c.Text
		for _, term := range terms {
			if term.Standard == "" {
				continue
			}
			for _, alias := range term.Aliases {
				if alias == "" || alias == term.Standard {
					continue
				}
				t = strings.ReplaceAll(t, alias, term.Standard)
			}
		}
		out[i] = Cue{Start: c.Start, End: c.End, Text: t}
	}
	return out
}
