package subtitle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTemp 写一个临时字幕文件并返回路径。
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return p
}

// TestParseSRT 校验 SRT 解析：时间戳换算、文本提取、去标签。
func TestParseSRT(t *testing.T) {
	src := "1\n00:00:01,500 --> 00:00:03,000\n你好<font>世界</font>\n\n2\n00:00:04,000 --> 00:00:05,250\n第二行\n换行文本\n"
	p := writeTemp(t, "a.srt", src)
	cues, err := ParseFile(p)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(cues) != 2 {
		t.Fatalf("期望 2 条，实际 %d", len(cues))
	}
	if cues[0].Start != 1.5 || cues[0].End != 3.0 {
		t.Fatalf("时间戳错误: %+v", cues[0])
	}
	if cues[0].Text != "你好世界" {
		t.Fatalf("去标签错误: %q", cues[0].Text)
	}
	if cues[1].Text != "第二行\n换行文本" {
		t.Fatalf("多行文本错误: %q", cues[1].Text)
	}
}

// TestParseVTT 校验 WebVTT 解析。
func TestParseVTT(t *testing.T) {
	src := "WEBVTT\n\n00:00:02.000 --> 00:00:04.000\nhello\n\n00:00:05.000 --> 00:00:06.000\nworld\n"
	p := writeTemp(t, "a.vtt", src)
	cues, err := ParseFile(p)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(cues) != 2 {
		t.Fatalf("期望 2 条，实际 %d", len(cues))
	}
	if cues[0].Start != 2.0 || cues[1].Text != "world" {
		t.Fatalf("VTT 解析错误: %+v", cues)
	}
}

// TestParseASS 校验 ASS/SSA 解析（按 Format 定位 Text 列，处理 \N 换行）。
func TestParseASS(t *testing.T) {
	src := "[Script Info]\nTitle: t\n\n[Events]\n" +
		"Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n" +
		"Dialogue: 0,0:00:01.00,0:00:03.50,Default,,0,0,0,,第一行\\N第二行\n" +
		"Dialogue: 0,0:00:04.00,0:00:05.00,Default,,0,0,0,,加粗文本\n"
	p := writeTemp(t, "a.ass", src)
	cues, err := ParseFile(p)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(cues) != 2 {
		t.Fatalf("期望 2 条，实际 %d", len(cues))
	}
	if cues[0].Start != 1.0 || cues[0].End != 3.5 {
		t.Fatalf("ASS 时间错误: %+v", cues[0])
	}
	if cues[0].Text != "第一行\n第二行" {
		t.Fatalf("ASS 换行未处理: %q", cues[0].Text)
	}
}

// TestParseTXT 校验纯文本整体作为一条 cue。
func TestParseTXT(t *testing.T) {
	p := writeTemp(t, "a.txt", "  一段纯文本  ")
	cues, err := ParseFile(p)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(cues) != 1 || cues[0].Text != "一段纯文本" {
		t.Fatalf("TXT 解析错误: %+v", cues)
	}
}

// TestFormatSRTRoundtrip 校验 FormatSRT 可被 parseSRT 回读。
func TestFormatSRTRoundtrip(t *testing.T) {
	in := []Cue{{Start: 1.5, End: 3.25, Text: "回读"}, {Start: 10, End: 12.001, Text: "第二条"}}
	out := FormatSRT(in)
	cues := parseSRT(out)
	if len(cues) != 2 {
		t.Fatalf("roundtrip 条数错误: %d", len(cues))
	}
	if cues[0].Start != 1.5 || cues[1].End != 12.001 || cues[1].Text != "第二条" {
		t.Fatalf("roundtrip 内容错误: %+v", cues)
	}
}

// TestParseDanmuXML 校验弹幕 XML <d> 提取。
func TestParseDanmuXML(t *testing.T) {
	src := "<i><d p=\"1,1,25\">弹幕一</d><d p=\"2\">弹幕二</d></i>"
	p := writeTemp(t, "a.xml", src)
	got, err := ParseDanmu(p)
	if err != nil {
		t.Fatalf("danmu: %v", err)
	}
	if len(got) != 2 || got[0] != "弹幕一" || got[1] != "弹幕二" {
		t.Fatalf("弹幕解析错误: %+v", got)
	}
}

// TestFusePriorityNativeWins 校验融合时 native 优先于 asr，asr 仅补空缺。
func TestFusePriorityNativeWins(t *testing.T) {
	native := []Cue{{Start: 0, End: 5, Text: "原生"}}
	asr := []Cue{{Start: 0, End: 3, Text: "语音"}}
	out := Fuse([]Source{{Kind: "native", Cues: native}, {Kind: "asr", Cues: asr}})
	if len(out) != 1 || out[0].Text != "原生" {
		t.Fatalf("native 应优先并去重 asr: %+v", out)
	}
}

// TestFuseASRGapFill 校验 native 未覆盖的 asr 分片会被补入。
func TestFuseASRGapFill(t *testing.T) {
	native := []Cue{{Start: 0, End: 2, Text: "原生开头"}}
	asr := []Cue{{Start: 30, End: 35, Text: "语音补充"}}
	out := Fuse([]Source{{Kind: "native", Cues: native}, {Kind: "asr", Cues: asr}})
	if len(out) != 2 {
		t.Fatalf("asr 空缺应补入，期望 2 条，实际 %d: %+v", len(out), out)
	}
}

// TestFuseVisionAppended 校验 vision 文本被追加并按时间排序。
func TestFuseVisionAppended(t *testing.T) {
	native := []Cue{{Start: 10, End: 12, Text: "字幕"}}
	vision := []Cue{{Start: 1, End: 2, Text: "画面文字"}}
	out := Fuse([]Source{{Kind: "native", Cues: native}, {Kind: "vision", Cues: vision}})
	if len(out) != 2 {
		t.Fatalf("vision 应追加，期望 2 条，实际 %d", len(out))
	}
	if out[0].Text != "画面文字" {
		t.Fatalf("应按时间排序，首条应为画面文字: %+v", out)
	}
}

// TestFuseDedupe 校验相同文本被去重。
func TestFuseDedupe(t *testing.T) {
	native := []Cue{{Start: 0, End: 1, Text: "重复"}, {Start: 2, End: 3, Text: "重复"}}
	out := Fuse([]Source{{Kind: "native", Cues: native}})
	if len(out) != 1 {
		t.Fatalf("重复文本应去重，期望 1 条，实际 %d", len(out))
	}
}

// TestApplyTerms 校验术语纠错且不修改原切片。
func TestApplyTerms(t *testing.T) {
	cues := []Cue{{Start: 0, End: 1, Text: "用 糖水 做 斯涅尔 演示"}}
	terms := []Term{{Standard: "斯涅尔定律", Aliases: []string{"斯涅尔"}}, {Standard: "糖水", Aliases: []string{"糖水"}}}
	out := ApplyTerms(cues, terms)
	if !strings.Contains(out[0].Text, "斯涅尔定律") {
		t.Fatalf("术语未替换: %q", out[0].Text)
	}
	if cues[0].Text != "用 糖水 做 斯涅尔 演示" {
		t.Fatalf("原切片被修改: %q", cues[0].Text)
	}
}

// TestApplyTermsEmpty 校验无术语时原样返回。
func TestApplyTermsEmpty(t *testing.T) {
	cues := []Cue{{Start: 0, End: 1, Text: "原文"}}
	out := ApplyTerms(cues, nil)
	if len(out) != 1 || out[0].Text != "原文" {
		t.Fatalf("无术语应原样返回: %+v", out)
	}
}
