package subtitle

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Cue is one subtitle line with a time range (seconds).
type Cue struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// ParseFile auto-detects the subtitle format by extension and parses it.
func ParseFile(path string) ([]Cue, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	content := string(data)
	switch ext {
	case ".srt":
		return parseSRT(content), nil
	case ".vtt":
		return parseVTT(content), nil
	case ".ass", ".ssa":
		return parseASS(content), nil
	case ".txt":
		return parseTXT(content), nil
	}
	// fall back by sniffing
	if strings.Contains(content, "[Events]") {
		return parseASS(content), nil
	}
	if strings.HasPrefix(strings.TrimSpace(content), "WEBVTT") {
		return parseVTT(content), nil
	}
	if strings.Contains(content, "-->") {
		return parseSRT(content), nil
	}
	return parseTXT(content), nil
}

var reSRTTime = regexp.MustCompile(`(\d{1,2}):(\d{2}):(\d{2})[,\.](\d{1,3})`)

func parseClock(s string) float64 {
	m := reSRTTime.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	h, _ := strconv.Atoi(m[1])
	mi, _ := strconv.Atoi(m[2])
	se, _ := strconv.Atoi(m[3])
	ms, _ := strconv.Atoi(padRight(m[4], 3))
	return float64(h*3600 + mi*60 + se) + float64(ms)/1000.0
}

func padRight(s string, n int) string {
	for len(s) < n {
		s += "0"
	}
	return s[:n]
}

func parseSRT(content string) []Cue {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	blocks := regexp.MustCompile(`\n\s*\n`).Split(strings.TrimSpace(content), -1)
	var cues []Cue
	for _, b := range blocks {
		lines := strings.Split(strings.TrimSpace(b), "\n")
		if len(lines) == 0 {
			continue
		}
		ti := -1
		for i, ln := range lines {
			if strings.Contains(ln, "-->") {
				ti = i
				break
			}
		}
		if ti < 0 {
			continue
		}
		parts := strings.SplitN(lines[ti], "-->", 2)
		start := parseClock(parts[0])
		end := 0.0
		if len(parts) > 1 {
			end = parseClock(parts[1])
		}
		text := strings.Join(lines[ti+1:], "\n")
		text = strings.TrimSpace(stripTags(text))
		if text != "" {
			cues = append(cues, Cue{Start: start, End: end, Text: text})
		}
	}
	return cues
}

func parseVTT(content string) []Cue {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var cues []Cue
	i := 0
	for i < len(lines) {
		ln := lines[i]
		if strings.Contains(ln, "-->") {
			parts := strings.SplitN(ln, "-->", 2)
			start := parseClock(parts[0])
			end := 0.0
			if len(parts) > 1 {
				end = parseClock(parts[1])
			}
			var txt []string
			i++
			for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
				txt = append(txt, lines[i])
				i++
			}
			t := strings.TrimSpace(stripTags(strings.Join(txt, "\n")))
			if t != "" {
				cues = append(cues, Cue{Start: start, End: end, Text: t})
			}
		} else {
			i++
		}
	}
	return cues
}

func parseASS(content string) []Cue {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	inEvents := false
	var cues []Cue
	textIdx := 9
	for _, ln := range lines {
		l := strings.TrimSpace(ln)
		if strings.HasPrefix(l, "[Events]") {
			inEvents = true
			continue
		}
		if !inEvents {
			continue
		}
		if strings.HasPrefix(l, "Format:") {
			fields := strings.Split(l[len("Format:"):], ",")
			for i, f := range fields {
				if strings.TrimSpace(strings.ToLower(f)) == "text" {
					textIdx = i
				}
			}
			continue
		}
		if strings.HasPrefix(l, "Dialogue:") {
			body := strings.TrimPrefix(l, "Dialogue:")
			parts := strings.SplitN(body, ",", textIdx+1)
			if len(parts) < textIdx+1 {
				continue
			}
			start := parseASSTime(parts[1])
			end := parseASSTime(parts[2])
			text := strings.TrimSpace(stripTags(parts[textIdx]))
			text = strings.ReplaceAll(text, "\\N", "\n")
			text = strings.ReplaceAll(text, "\\n", "\n")
			if text != "" {
				cues = append(cues, Cue{Start: start, End: end, Text: text})
			}
		}
	}
	return cues
}

func parseASSTime(s string) float64 {
	// H:MM:SS.cc
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	sec, _ := strconv.ParseFloat(parts[2], 64)
	return float64(h*3600+m*60) + sec
}

func parseTXT(content string) []Cue {
	t := strings.TrimSpace(content)
	if t == "" {
		return nil
	}
	return []Cue{{Start: 0, End: 0, Text: t}}
}

var reTag = regexp.MustCompile(`<[^>]+>`)

func stripTags(s string) string {
	return reTag.ReplaceAllString(s, "")
}

// FormatSRT renders cues to SRT text.
func FormatSRT(cues []Cue) string {
	var b strings.Builder
	for i, c := range cues {
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n\n", i+1,
			fmtStamp(c.Start), fmtStamp(c.End), c.Text)
	}
	return b.String()
}

func fmtStamp(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	ms := int(sec*1000 + 0.5)
	h := ms / 3600000
	ms -= h * 3600000
	m := ms / 60000
	ms -= m * 60000
	s := ms / 1000
	ms -= s * 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

// ParseDanmu extracts raw comment texts from a danmu file (XML <d>...</d> or plain text lines).
func ParseDanmu(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	var out []string
	if strings.Contains(content, "<d ") || strings.Contains(content, "<d>") {
		re := regexp.MustCompile(`(?s)<d[^>]*>(.*?)</d>`)
		for _, m := range re.FindAllStringSubmatch(content, -1) {
			t := strings.TrimSpace(stripTags(m[1]))
			if t != "" {
				out = append(out, t)
			}
		}
		return out, nil
	}
	for _, ln := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(ln)
		if t != "" {
			out = append(out, t)
		}
	}
	return out, nil
}
