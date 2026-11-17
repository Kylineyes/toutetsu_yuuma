package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/config"
	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/db"
	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/deepseek"
	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/media"
	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/subtitle"
)

// Runner executes the analysis pipeline for one task.
type Runner struct {
	DB  *db.DB
	Cfg *config.Config
}

// Cost accumulates usage for display (never used to abort).
type Cost struct {
	ASRSeconds       float64 `json:"asr_seconds"`
	VisionImages     int     `json:"vision_images"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
}

func (r *Runner) setState(id, status, stage string, progress int, errMsg string) {
	_ = r.DB.UpdateTaskState(id, status, stage, progress, errMsg)
}

func (r *Runner) fail(id, stage string, err error) {
	r.setState(id, "failed", stage, 0, truncateMsg(err.Error()))
}

func truncateMsg(s string) string {
	if len(s) > 500 {
		return s[:500]
	}
	return s
}

func (r *Runner) clientFor(profileID int64) (*deepseek.Client, error) {
	if profileID <= 0 {
		return nil, fmt.Errorf("未选择配置方案 (profile_id=%d)", profileID)
	}
	p, err := r.DB.GetProfile(profileID)
	if err != nil {
		return nil, fmt.Errorf("读取方案失败: %v", err)
	}
	return deepseek.New(p.BaseURL, p.APIKey, p.Model), nil
}

func (r *Runner) withRetry(fn func() error) error {
	var err error
	for i := 0; i < 3; i++ {
		if err = fn(); err == nil {
			return nil
		}
		time.Sleep(time.Duration(1<<i) * time.Second)
	}
	return err
}

// Run processes one task synchronously (called from a worker goroutine).
func (r *Runner) Run(ctx context.Context, taskID string) {
	task, err := r.DB.GetTask(taskID)
	if err != nil {
		return
	}
	defer func() {
		if rec := recover(); rec != nil {
			r.setState(taskID, "failed", "panic", 0, fmt.Sprintf("panic: %v", rec))
		}
	}()

	baseDir := filepath.Join(r.Cfg.DataDir, "task", taskID)
	uploadDir := filepath.Join(baseDir, "upload")
	subsDir := filepath.Join(baseDir, "subs")
	outDir := filepath.Join(baseDir, "output")
	workDir := filepath.Join(r.Cfg.TmpDir, "intermediates", taskID)
	for _, d := range []string{uploadDir, subsDir, outDir, workDir} {
		_ = os.MkdirAll(d, 0o755)
	}

	r.setState(taskID, "running", "download", 5, "")

	// 1. video
	videoPath := task.VideoPath
	if task.SourceType == "url" && task.VideoURL != "" {
		vp := filepath.Join(uploadDir, "video")
		if err := r.withRetry(func() error { return downloadFile(ctx, task.VideoURL, vp) }); err != nil {
			r.fail(taskID, "download", err)
			return
		}
		videoPath = vp
		_ = r.DB.UpdateTaskVideo(taskID, vp)
	}
	if videoPath == "" {
		r.fail(taskID, "video", fmt.Errorf("没有视频源"))
		return
	}

	// 2. audio
	r.setState(taskID, "running", "audio", 15, "")
	audioWav := filepath.Join(workDir, "audio.wav")
	if err := r.withRetry(func() error { return media.ExtractAudio(ctx, videoPath, audioWav) }); err != nil {
		r.fail(taskID, "audio", err)
		return
	}
	dur, _ := media.ProbeDuration(ctx, videoPath)

	// uploaded subtitles
	subs, _ := r.DB.ListTaskSubtitles(taskID)
	var uploadedNative, uploadedAI, uploadedVision []subtitle.Cue
	var danmuRaw []string
	for _, s := range subs {
		switch s.Kind {
		case "native":
			if c, err := subtitle.ParseFile(s.StoredPath); err == nil {
				uploadedNative = append(uploadedNative, c...)
			}
		case "ai":
			if c, err := subtitle.ParseFile(s.StoredPath); err == nil {
				uploadedAI = append(uploadedAI, c...)
			}
		case "vision":
			if c, err := subtitle.ParseFile(s.StoredPath); err == nil {
				uploadedVision = append(uploadedVision, c...)
			}
		case "danmu":
			if t, err := subtitle.ParseDanmu(s.StoredPath); err == nil {
				danmuRaw = append(danmuRaw, t...)
			}
		}
	}

	cost := &Cost{}

	// 3. ASR
	var asrCues []subtitle.Cue
	needASR := task.ASRProfileID > 0 && (task.GenerateAIViaASR || (len(uploadedNative) == 0 && len(uploadedAI) == 0))
	if needASR {
		r.setState(taskID, "running", "asr", 30, "")
		chunks, cutErr := media.SplitAudio(ctx, audioWav, workDir, dur)
		if cutErr == nil {
			asrClient, e2 := r.clientFor(task.ASRProfileID)
			if e2 != nil {
				r.fail(taskID, "asr", e2)
				return
			}
			for _, ch := range chunks {
				var res *deepseek.ASRResult
				e := r.withRetry(func() error {
					rr, err := asrClient.Transcribe(ctx, ch.Path)
					if err != nil {
						return err
					}
					res = rr
					return nil
				})
				if e != nil || res == nil {
					continue
				}
				if strings.TrimSpace(res.Text) != "" {
					asrCues = append(asrCues, subtitle.Cue{Start: ch.Start, End: ch.End, Text: res.Text})
					cost.ASRSeconds += (ch.End - ch.Start)
				}
			}
			if len(asrCues) > 0 {
				p := filepath.Join(subsDir, "asr.srt")
				_ = os.WriteFile(p, []byte(subtitle.FormatSRT(asrCues)), 0o644)
				_ = r.DB.AddTaskSubtitle(&db.TaskSubtitle{TaskID: taskID, Kind: "asr", OrigName: "asr.srt", StoredPath: p, Generated: true})
			}
		}
	}

	// 4/5. frames + vision subtitle
	var visionCues []subtitle.Cue
	needVision := task.VisionProfileID > 0 && task.GenerateVision
	var frames []media.Frame
	if (needVision || task.ReadingProfileID > 0) && task.ScreenshotMode != "off" {
		r.setState(taskID, "running", "frame", 45, "")
		frames, _ = media.ExtractFrames(ctx, videoPath, filepath.Join(workDir, "frames"), task.ScreenshotInterval, r.Cfg.MaxFrames, dur)
	}
	if needVision && len(frames) > 0 {
		r.setState(taskID, "running", "vision", 52, "")
		vClient, e := r.clientFor(task.VisionProfileID)
		if e == nil {
			for _, fr := range frames {
				var res *deepseek.ChatResult
				er := r.withRetry(func() error {
					rr, err := vClient.Vision(ctx, "识别这张视频截图中的字幕/画面文字与关键信息，只输出文字内容；若无文字则输出空。", []string{fr.Path})
					if err != nil {
						return err
					}
					res = rr
					return nil
				})
				if er != nil || res == nil {
					continue
				}
				cost.VisionImages++
				cost.PromptTokens += res.PromptTokens
				cost.CompletionTokens += res.CompletionTokens
				txt := strings.TrimSpace(res.Text)
				if txt != "" && txt != "空" {
					visionCues = append(visionCues, subtitle.Cue{Start: fr.Time, End: fr.Time + float64(task.ScreenshotInterval), Text: txt})
				}
			}
			if len(visionCues) > 0 {
				p := filepath.Join(subsDir, "vision.srt")
				_ = os.WriteFile(p, []byte(subtitle.FormatSRT(visionCues)), 0o644)
				_ = r.DB.AddTaskSubtitle(&db.TaskSubtitle{TaskID: taskID, Kind: "vision", OrigName: "vision.srt", StoredPath: p, Generated: true})
			}
		}
	}

	// 6. fuse
	r.setState(taskID, "running", "fuse", 65, "")
	var sources []subtitle.Source
	if len(uploadedNative) > 0 {
		sources = append(sources, subtitle.Source{Kind: "native", Cues: uploadedNative})
	}
	if len(uploadedAI) > 0 {
		sources = append(sources, subtitle.Source{Kind: "ai", Cues: uploadedAI})
	}
	if len(asrCues) > 0 {
		sources = append(sources, subtitle.Source{Kind: "asr", Cues: asrCues})
	}
	if len(uploadedVision) > 0 {
		sources = append(sources, subtitle.Source{Kind: "vision", Cues: uploadedVision})
	}
	if len(visionCues) > 0 {
		sources = append(sources, subtitle.Source{Kind: "vision", Cues: visionCues})
	}
	fused := subtitle.Fuse(sources)
	fused = subtitle.ApplyTerms(fused, r.loadTaskTerms(taskID))

	fusedPath := filepath.Join(subsDir, "fused.srt")
	_ = os.WriteFile(fusedPath, []byte(subtitle.FormatSRT(fused)), 0o644)
	unifiedText := cuesToText(fused)

	danmuText := ""
	if len(danmuRaw) > 0 {
		danmuText = SummarizeDanmu(CleanDanmu(danmuRaw), 200)
	}

	// 7. summary + knowledge (T rounds)
	r.setState(taskID, "running", "summary", 75, "")
	summaryMD, knowledgeMD, err := r.generate(ctx, task, unifiedText, danmuText, frames, workDir, int(dur), cost)
	if err != nil {
		r.fail(taskID, "summary", err)
		return
	}
	_ = os.WriteFile(filepath.Join(outDir, "summary.md"), []byte(summaryMD), 0o644)
	_ = os.WriteFile(filepath.Join(outDir, "knowledge.md"), []byte(knowledgeMD), 0o644)

	cj, _ := json.Marshal(cost)
	_ = r.DB.UpdateTaskCost(taskID, string(cj))
	r.setState(taskID, "done", "done", 100, "")
}

func (r *Runner) loadTaskTerms(taskID string) []subtitle.Term {
	kbs, _ := r.DB.ListTaskKBs(taskID)
	var terms []subtitle.Term
	for _, k := range kbs { // ordered by priority asc => front wins (append later has lower priority)
		v, err := r.DB.GetKBVersion(k.KBID, k.Version)
		if err != nil {
			continue
		}
		t, _, err := LoadTerms(v.DirPath)
		if err == nil {
			terms = append(terms, t...)
		}
	}
	return terms
}

type readingQ struct {
	Timestamp float64 `json:"timestamp"`
	Question  string  `json:"question"`
}

func (r *Runner) generate(ctx context.Context, task *db.Task, unified, danmu string, frames []media.Frame, workDir string, dur int, cost *Cost) (string, string, error) {
	sClient, err := r.clientFor(task.SummaryProfileID)
	if err != nil {
		return "", "", err
	}
	var rClient *deepseek.Client
	if task.ReadingProfileID > 0 {
		if c, e := r.clientFor(task.ReadingProfileID); e == nil {
			rClient = c
		}
	}

	ctxText := "视频字幕（统一稿）：\n" + unified
	if danmu != "" {
		ctxText += "\n\n观众弹幕摘要（可能含噪声，仅作热点/反馈参考）：\n" + danmu
	}
	if task.SummaryPreference != "" {
		ctxText += "\n\n总结方向偏好：\n" + task.SummaryPreference
	}

	T := task.ReadingDepth
	if T < 0 {
		T = 0
	}
	for round := 1; round <= T; round++ {
		if rClient == nil || len(frames) == 0 {
			break
		}
		qPrompt := ctxText + "\n\n你正在进行第" + strconv.Itoa(round) + "轮深度阅读。请基于目前对字幕的理解，列出你想回看视频画面的时间点与疑问。严格输出 JSON 数组，元素形如 {\"timestamp\": 秒数, \"question\": \"...\"}，最多 8 条。"
		res, e := sClient.Chat(ctx, []deepseek.Message{{Role: "user", Content: qPrompt}})
		if e != nil {
			break
		}
		cost.PromptTokens += res.PromptTokens
		cost.CompletionTokens += res.CompletionTokens
		qs := parseQuestions(res.Text)
		if len(qs) == 0 {
			break
		}
		turnDir := filepath.Join(workDir, "turn", strconv.Itoa(round))
		_ = os.MkdirAll(turnDir, 0o755)
		var roundFindings []string
		for i, q := range qs {
			near := nearestFrames(frames, q.Timestamp, float64(task.ScreenshotInterval))
			if len(near) == 0 {
				continue
			}
			ans, e := rClient.Vision(ctx, q.Question+"\n请结合这几帧画面回答。", near)
			if e != nil {
				continue
			}
			cost.VisionImages += len(near)
			cost.PromptTokens += ans.PromptTokens
			cost.CompletionTokens += ans.CompletionTokens
			roundFindings = append(roundFindings, fmt.Sprintf("[轮%d/%d @%.0fs] Q:%s\nA:%s", round, i+1, q.Timestamp, q.Question, strings.TrimSpace(ans.Text)))
		}
		if len(roundFindings) == 0 {
			break
		}
		_ = os.WriteFile(filepath.Join(turnDir, "questions.json"), []byte(res.Text), 0o644)
		_ = os.WriteFile(filepath.Join(turnDir, "answers.txt"), []byte(strings.Join(roundFindings, "\n\n")), 0o644)
		ctxText += "\n\n[第" + strconv.Itoa(round) + "轮回看结论]\n" + strings.Join(roundFindings, "\n")
	}

	title := strings.TrimSuffix(task.Filename, filepath.Ext(task.Filename))
	finalPrompt := ctxText + fmt.Sprintf("\n\n请基于以上信息产出两份 Markdown，严格用 JSON 返回：{\"summary_md\":\"...\",\"knowledge_md\":\"...\"}。\n"+
		"summary_md 结构：\n# %s\n- 视频：%s\n- 时长：%s\n- 任务ID：%s\n- 分析时间：%s\n## 分析概述\n（仿论文摘要，3-6 句，概述全片、重点与结论）\n## 分析具体内容\n### 主题一：<该段主题>\n（要点，附时间戳）\n（继续按主题分节，每节写明主题）\n"+
		"knowledge_md 结构：知识点卡片条目化，每条=标题+一句话结论+详细说明(2-5行)+关键词/标签+来源时间戳，顶部带目录。\n只输出 JSON，不要多余文字。",
		title, task.Filename, mmss(dur), task.ID, time.Now().Format("2006-01-02 15:04"))

	finalRes, err := sClient.Chat(ctx, []deepseek.Message{{Role: "user", Content: finalPrompt}})
	if err != nil {
		return "", "", err
	}
	cost.PromptTokens += finalRes.PromptTokens
	cost.CompletionTokens += finalRes.CompletionTokens
	summary, kn := parseFinal(finalRes.Text)
	if strings.TrimSpace(summary) == "" {
		summary = defaultSummary(task, unified, dur)
	}
	if strings.TrimSpace(kn) == "" {
		kn = "# 知识点\n\n（未能解析出结构化知识点）"
	}
	return summary, kn, nil
}

func parseQuestions(s string) []readingQ {
	s = stripFences(s)
	lo := strings.Index(s, "[")
	hi := strings.LastIndex(s, "]")
	if lo < 0 || hi <= lo {
		return nil
	}
	var qs []readingQ
	if json.Unmarshal([]byte(s[lo:hi+1]), &qs) != nil {
		return nil
	}
	return qs
}

func parseFinal(s string) (string, string) {
	s = stripFences(s)
	lo := strings.Index(s, "{")
	hi := strings.LastIndex(s, "}")
	if lo < 0 || hi <= lo {
		return "", ""
	}
	var out struct {
		SummaryMD   string `json:"summary_md"`
		KnowledgeMD string `json:"knowledge_md"`
	}
	if json.Unmarshal([]byte(s[lo:hi+1]), &out) != nil {
		return "", ""
	}
	return out.SummaryMD, out.KnowledgeMD
}

func stripFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if i := strings.Index(s, "\n"); i >= 0 {
			s = s[i+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	}
	return strings.TrimSpace(s)
}

func nearestFrames(frames []media.Frame, ts, interval float64) []string {
	var out []string
	for _, f := range frames {
		if abs(f.Time-ts) <= interval+1 {
			out = append(out, f.Path)
			if len(out) >= 3 {
				break
			}
		}
	}
	if len(out) == 0 && len(frames) > 0 {
		best := frames[0]
		for _, f := range frames {
			if abs(f.Time-ts) < abs(best.Time-ts) {
				best = f
			}
		}
		out = append(out, best.Path)
	}
	return out
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func cuesToText(cues []subtitle.Cue) string {
	var b strings.Builder
	for _, c := range cues {
		if strings.TrimSpace(c.Text) == "" {
			continue
		}
		fmt.Fprintf(&b, "[%s] %s\n", mmss(int(c.Start)), strings.ReplaceAll(c.Text, "\n", " "))
	}
	return b.String()
}

func mmss(sec int) string {
	if sec < 0 {
		sec = 0
	}
	return fmt.Sprintf("%02d:%02d", sec/60, sec%60)
}

func defaultSummary(task *db.Task, unified string, dur int) string {
	title := strings.TrimSuffix(task.Filename, filepath.Ext(task.Filename))
	if len(unified) > 4000 {
		unified = unified[:4000]
	}
	return fmt.Sprintf("# %s\n\n- 视频：%s\n- 时长：%s\n- 任务ID：%s\n- 分析时间：%s\n\n## 分析概述\n（AI 未返回结构化总结，以下为统一稿节选，请检查模型输出）\n\n## 分析具体内容\n\n%s\n",
		title, task.Filename, mmss(dur), task.ID, time.Now().Format("2006-01-02 15:04"), unified)
}

func downloadFile(ctx context.Context, url, dst string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 toutetsu_yuuma")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("下载视频 HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
