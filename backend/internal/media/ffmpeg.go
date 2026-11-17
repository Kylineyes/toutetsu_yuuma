package media

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Chunk is one ASR-able audio segment (kept <=120s, with overlap between neighbors).
type Chunk struct {
	Index int
	Path  string
	Start float64
	End   float64
}

// Frame is a sampled screenshot with its timestamp.
type Frame struct {
	Index int
	Path  string
	Time  float64
}

// ProbeDuration returns media duration in seconds.
func ProbeDuration(ctx context.Context, file string) (float64, error) {
	out, err := exec.CommandContext(ctx, "ffprobe", "-v", "error",
		"-show_entries", "format=duration", "-of", "csv=p=0", file).Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe: %v", err)
	}
	s := strings.TrimSpace(string(out))
	d, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("ffprobe parse %q: %v", s, err)
	}
	return d, nil
}

// ExtractAudio extracts a 16k mono PCM WAV track (format required by DeepSeek ASR).
func ExtractAudio(ctx context.Context, video, outWav string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", video,
		"-vn", "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le", outWav)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg extract audio: %v: %s", err, tail(string(out), 400))
	}
	return nil
}

var reSilStart = regexp.MustCompile(`silence_start: ([0-9.]+)`)
var reSilEnd = regexp.MustCompile(`silence_end: ([0-9.]+)`)

// detectSilence returns midpoints of silent intervals (seconds).
func detectSilence(ctx context.Context, wav string) []float64 {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", wav,
		"-af", "silencedetect=noise=-30dB:d=0.5", "-f", "null", "-")
	out, _ := cmd.CombinedOutput()
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	var mids []float64
	var curStart float64 = -1
	for sc.Scan() {
		line := sc.Text()
		if m := reSilStart.FindStringSubmatch(line); m != nil {
			curStart, _ = strconv.ParseFloat(m[1], 64)
		}
		if m := reSilEnd.FindStringSubmatch(line); m != nil {
			e, _ := strconv.ParseFloat(m[1], 64)
			if curStart >= 0 {
				mids = append(mids, (curStart+e)/2)
			}
			curStart = -1
		}
	}
	return mids
}

const (
	targetChunk = 110.0 // <=120s hard limit, keep margin
	overlapSec  = 4.0
)

// SplitAudio slices the WAV into <=120s chunks, snapping cut points to nearby silence.
func SplitAudio(ctx context.Context, wav, workDir string, duration float64) ([]Chunk, error) {
	if duration <= 0 {
		d, err := ProbeDuration(ctx, wav)
		if err != nil {
			return nil, err
		}
		duration = d
	}
	mids := detectSilence(ctx, wav)
	pickCut := func(target float64) float64 {
		best := -1.0
		for _, m := range mids {
			if m > target-12 && m <= target {
				if best < 0 || (target-m) < (target-best) {
					best = m
				}
			}
		}
		if best > 0 {
			return best
		}
		return target
	}

	var ranges [][2]float64
	if duration <= 120 {
		ranges = append(ranges, [2]float64{0, duration})
	} else {
		start := 0.0
		for start < duration {
			end := start + targetChunk
			if end >= duration {
				end = duration
			} else {
				end = pickCut(end)
				if end <= start+5 {
					end = start + targetChunk
				}
			}
			ranges = append(ranges, [2]float64{start, end})
			if end >= duration {
				break
			}
			start = end - overlapSec
		}
	}

	var chunks []Chunk
	for i, r := range ranges {
		p := filepath.Join(workDir, fmt.Sprintf("chunk_%03d.wav", i))
		dur := r[1] - r[0]
		cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-ss", ftoa(r[0]), "-t", ftoa(dur),
			"-i", wav, "-c", "copy", p)
		if out, err := cmd.CombinedOutput(); err != nil {
			// fallback: re-encode
			cmd = exec.CommandContext(ctx, "ffmpeg", "-y", "-ss", ftoa(r[0]), "-t", ftoa(dur),
				"-i", wav, "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le", p)
			if out2, err2 := cmd.CombinedOutput(); err2 != nil {
				return nil, fmt.Errorf("split chunk %d: %v: %s", i, err2, tail(string(out2), 300))
			}
			_ = out
		}
		chunks = append(chunks, Chunk{Index: i, Path: p, Start: r[0], End: r[1]})
	}
	return chunks, nil
}

// ExtractFrames samples screenshots at a fixed interval (simple mode), capped at maxFrames.
func ExtractFrames(ctx context.Context, video, outDir string, intervalSeconds, maxFrames int, duration float64) ([]Frame, error) {
	if intervalSeconds <= 0 {
		intervalSeconds = 60
	}
	if maxFrames <= 0 {
		maxFrames = 120
	}
	os.MkdirAll(outDir, 0o755)
	var frames []Frame
	idx := 0
	for t := 0.0; ; t += float64(intervalSeconds) {
		if duration > 0 && t >= duration {
			break
		}
		if idx >= maxFrames {
			break
		}
		p := filepath.Join(outDir, fmt.Sprintf("frame_%05d.jpg", idx))
		cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-ss", ftoa(t), "-i", video,
			"-frames:v", "1", "-q:v", "2", p)
		if out, err := cmd.CombinedOutput(); err != nil {
			_ = out
			idx++
			continue
		}
		if _, err := os.Stat(p); err == nil {
			frames = append(frames, Frame{Index: idx, Path: p, Time: t})
		}
		idx++
		if duration <= 0 && idx > maxFrames {
			break
		}
	}
	return frames, nil
}

func ftoa(f float64) string { return strconv.FormatFloat(f, 'f', 3, 64) }

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
