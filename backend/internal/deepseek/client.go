package deepseek

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client talks to one provider using a chosen protocol. One Client maps to one Profile.
type Client struct {
	BaseURL     string
	APIKey      string
	Model       string
	Protocol    string
	Temperature *float64
	HTTP        *http.Client
}

// New builds a Client. protocol defaults to openai_chat; timeoutSeconds<=0 uses 180s.
func New(baseURL, apiKey, model, protocol string, temperature *float64, timeoutSeconds int) *Client {
	if protocol == "" {
		protocol = "openai_chat"
	}
	t := 180
	if timeoutSeconds > 0 {
		t = timeoutSeconds
	}
	return &Client{
		BaseURL:     strings.TrimRight(baseURL, "/"),
		APIKey:      apiKey,
		Model:       model,
		Protocol:    protocol,
		Temperature: temperature,
		HTTP:        &http.Client{Timeout: time.Duration(t) * time.Second},
	}
}

func (c *Client) url(p string) string { return c.BaseURL + p }

// ---- ASR ----

type ASRResult struct {
	Text       string `json:"text"`
	DurationMs int    `json:"duration_ms"`
}

// Transcribe dispatches by protocol (deepseek_asr / openai_audio).
func (c *Client) Transcribe(ctx context.Context, audioPath string) (*ASRResult, error) {
	switch c.Protocol {
	case "openai_audio", "openai_whisper":
		return c.transcribeOpenAI(ctx, audioPath)
	default:
		return c.transcribeDeepSeek(ctx, audioPath)
	}
}

func (c *Client) transcribeDeepSeek(ctx context.Context, audioPath string) (*ASRResult, error) {
	body, ct, err := multipartFile("file", audioPath, nil)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/v1/tools/asr"), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", ct)
	return c.doASR(req)
}

func (c *Client) transcribeOpenAI(ctx context.Context, audioPath string) (*ASRResult, error) {
	body, ct, err := multipartFile("file", audioPath, map[string]string{"model": c.Model})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/v1/audio/transcriptions"), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", ct)
	return c.doASR(req)
}

func (c *Client) doASR(req *http.Request) (*ASRResult, error) {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("asr http %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}
	var out ASRResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("asr decode: %v", err)
	}
	return &out, nil
}

// ---- chat / vision ----

// Part is one content part of a user message.
type Part struct {
	Type      string // text | image
	Text      string
	ImagePath string
}

type ChatResult struct {
	Text             string `json:"text"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
}

// ChatText sends a single text prompt.
func (c *Client) ChatText(ctx context.Context, text string) (*ChatResult, error) {
	return c.doChat(ctx, []Part{{Type: "text", Text: text}})
}

// Vision sends a prompt plus one or more local images.
func (c *Client) Vision(ctx context.Context, prompt string, imagePaths []string) (*ChatResult, error) {
	parts := []Part{{Type: "text", Text: prompt}}
	for _, p := range imagePaths {
		parts = append(parts, Part{Type: "image", ImagePath: p})
	}
	return c.doChat(ctx, parts)
}

func (c *Client) doChat(ctx context.Context, parts []Part) (*ChatResult, error) {
	switch c.Protocol {
	case "openai_responses":
		return c.chatResponses(ctx, parts)
	case "anthropic_messages":
		return c.chatAnthropic(ctx, parts)
	case "gemini":
		return c.chatGemini(ctx, parts)
	default:
		return c.chatOpenAI(ctx, parts)
	}
}

func (c *Client) tempJSON() map[string]any {
	m := map[string]any{}
	if c.Temperature != nil {
		m["temperature"] = *c.Temperature
	}
	return m
}

// --- OpenAI Chat Completions ---
func (c *Client) chatOpenAI(ctx context.Context, parts []Part) (*ChatResult, error) {
	content := []any{}
	for _, p := range parts {
		if p.Type == "text" {
			content = append(content, map[string]any{"type": "text", "text": p.Text})
		} else {
			b64, mime, err := loadImage(p.ImagePath)
			if err != nil {
				return nil, err
			}
			content = append(content, map[string]any{"type": "image_url",
				"image_url": map[string]any{"url": fmt.Sprintf("data:%s;base64,%s", mime, b64)}})
		}
	}
	payload := map[string]any{"model": c.Model, "messages": []any{map[string]any{"role": "user", "content": content}}}
	for k, v := range c.tempJSON() {
		payload[k] = v
	}
	raw, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/chat/completions"), bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	body, err := c.send(req)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			P int `json:"prompt_tokens"`
			C int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("openai_chat 响应异常: %s", truncate(string(body), 200))
	}
	return &ChatResult{Text: parsed.Choices[0].Message.Content, PromptTokens: parsed.Usage.P, CompletionTokens: parsed.Usage.C}, nil
}

// --- OpenAI Responses ---
func (c *Client) chatResponses(ctx context.Context, parts []Part) (*ChatResult, error) {
	content := []any{}
	for _, p := range parts {
		if p.Type == "text" {
			content = append(content, map[string]any{"type": "input_text", "text": p.Text})
		} else {
			b64, mime, err := loadImage(p.ImagePath)
			if err != nil {
				return nil, err
			}
			content = append(content, map[string]any{"type": "input_image", "image_url": fmt.Sprintf("data:%s;base64,%s", mime, b64)})
		}
	}
	payload := map[string]any{"model": c.Model, "input": []any{map[string]any{"role": "user", "content": content}}}
	for k, v := range c.tempJSON() {
		payload[k] = v
	}
	raw, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/responses"), bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	body, err := c.send(req)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		Usage struct {
			Input  int `json:"input_tokens"`
			Output int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("openai_responses 响应异常: %s", truncate(string(body), 200))
	}
	text := parsed.OutputText
	if text == "" {
		var b strings.Builder
		for _, o := range parsed.Output {
			for _, ct := range o.Content {
				b.WriteString(ct.Text)
			}
		}
		text = b.String()
	}
	return &ChatResult{Text: text, PromptTokens: parsed.Usage.Input, CompletionTokens: parsed.Usage.Output}, nil
}

// --- Anthropic Messages ---
func (c *Client) chatAnthropic(ctx context.Context, parts []Part) (*ChatResult, error) {
	content := []any{}
	for _, p := range parts {
		if p.Type == "text" {
			content = append(content, map[string]any{"type": "text", "text": p.Text})
		} else {
			b64, mime, err := loadImage(p.ImagePath)
			if err != nil {
				return nil, err
			}
			content = append(content, map[string]any{"type": "image",
				"source": map[string]any{"type": "base64", "media_type": mime, "data": b64}})
		}
	}
	payload := map[string]any{"model": c.Model, "max_tokens": 4096,
		"messages": []any{map[string]any{"role": "user", "content": content}}}
	for k, v := range c.tempJSON() {
		payload[k] = v
	}
	raw, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/v1/messages"), bytes.NewReader(raw))
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	body, err := c.send(req)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			Input  int `json:"input_tokens"`
			Output int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("anthropic 响应异常: %s", truncate(string(body), 200))
	}
	var b strings.Builder
	for _, ct := range parsed.Content {
		b.WriteString(ct.Text)
	}
	return &ChatResult{Text: b.String(), PromptTokens: parsed.Usage.Input, CompletionTokens: parsed.Usage.Output}, nil
}

// --- Gemini ---
func (c *Client) chatGemini(ctx context.Context, parts []Part) (*ChatResult, error) {
	gparts := []any{}
	for _, p := range parts {
		if p.Type == "text" {
			gparts = append(gparts, map[string]any{"text": p.Text})
		} else {
			b64, mime, err := loadImage(p.ImagePath)
			if err != nil {
				return nil, err
			}
			gparts = append(gparts, map[string]any{"inline_data": map[string]any{"mime_type": mime, "data": b64}})
		}
	}
	gen := map[string]any{}
	if c.Temperature != nil {
		gen["temperature"] = *c.Temperature
	}
	payload := map[string]any{"contents": []any{map[string]any{"parts": gparts}}}
	if len(gen) > 0 {
		payload["generationConfig"] = gen
	}
	raw, _ := json.Marshal(payload)
	u := c.url("/v1beta/models/" + c.Model + ":generateContent?key=" + c.APIKey)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	body, err := c.send(req)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Candidates) == 0 {
		return nil, fmt.Errorf("gemini 响应异常: %s", truncate(string(body), 200))
	}
	var b strings.Builder
	for _, pt := range parsed.Candidates[0].Content.Parts {
		b.WriteString(pt.Text)
	}
	return &ChatResult{Text: b.String(), PromptTokens: parsed.UsageMetadata.PromptTokenCount, CompletionTokens: parsed.UsageMetadata.CandidatesTokenCount}, nil
}

func (c *Client) send(req *http.Request) ([]byte, error) {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	return body, nil
}

// ---- helpers ----

func loadImage(path string) (b64 string, mime string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		mime = "image/png"
	case ".webp":
		mime = "image/webp"
	case ".gif":
		mime = "image/gif"
	default:
		mime = "image/jpeg"
	}
	return base64.StdEncoding.EncodeToString(data), mime, nil
}

func multipartFile(field, path string, extra map[string]string) (*bytes.Buffer, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range extra {
		mw.WriteField(k, v)
	}
	part, err := mw.CreateFormFile(field, filepath.Base(path))
	if err != nil {
		return nil, "", err
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, "", err
	}
	mw.Close()
	return &buf, mw.FormDataContentType(), nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
