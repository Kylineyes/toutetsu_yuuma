package pipeline

import (
	"regexp"
	"strings"
	"unicode"
)

var meaninglessDanmu = map[string]bool{
	"666": true, "6666": true, "66666": true, "233": true, "2333": true, "23333": true,
	"哈哈": true, "哈哈哈": true, "哈哈哈哈": true, "hhh": true, "hh": true,
	"awsl": true, "打卡": true, "沙发": true, "前排": true, "来了": true,
	"好": true, "赞": true, "顶": true, "1": true, "？": true, "?": true, "...": true,
	"第一": true, "首赞": true, "空降": true, "火钳刘明": true, "mark": true,
}

var reOnlySymbols = regexp.MustCompile(`^[\p{P}\p{S}\s]+$`)

func isMeaningless(s string) bool {
	t := strings.TrimSpace(s)
	if t == "" {
		return true
	}
	runes := []rune(t)
	if len(runes) < 2 {
		return true
	}
	lower := strings.ToLower(t)
	if meaninglessDanmu[lower] {
		return true
	}
	// all digits
	allDigit := true
	for _, r := range runes {
		if !unicode.IsDigit(r) {
			allDigit = false
			break
		}
	}
	if allDigit {
		return true
	}
	// pure symbols/punctuation
	if reOnlySymbols.MatchString(t) {
		return true
	}
	// single repeated rune (e.g. "aaaa", "！！！！")
	allSame := true
	for _, r := range runes {
		if r != runes[0] {
			allSame = false
			break
		}
	}
	if allSame {
		return true
	}
	return false
}

// CleanDanmu removes meaningless comments (spam / pure symbols / repeated chars) and dedupes.
func CleanDanmu(raw []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range raw {
		t := strings.TrimSpace(s)
		if isMeaningless(t) {
			continue
		}
		if seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// SummarizeDanmu renders cleaned comments into a compact context block (capped).
func SummarizeDanmu(kept []string, capN int) string {
	if capN <= 0 {
		capN = 200
	}
	if len(kept) > capN {
		kept = kept[:capN]
	}
	return strings.Join(kept, " / ")
}
