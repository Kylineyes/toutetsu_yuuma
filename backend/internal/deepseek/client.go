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

// Client talks to a DeepSeek-compatible endpoint. One Client maps to one Profile.
type Client struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

// New builds a Client with a sane default HTTP timeout.
func New(baseURL, apiKey, model string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		Model:   model,
		HTTP:    &http.Client{Timeout: 180 * time.Second},
	}
}

func (c *Client) url(path string) string {
	return c.BaseURL + path
}

// ---- ASR ----

// ASRResult holds the transcript for one audio chunk.
type ASRResult struct {
	Text       string `json:"text"`
	DurationMs int    `json:"duration_ms"`
}

// Transcribe sends one WAV/Opus file (<=120s, <=8MiB) to the ASR endpoint.
func (c *Client) Transcribe(ctx context.Context, audioPath string) (*ASRResult, error) {
	f, err := os.Open(audioPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, err
	}
	mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/v1/tools/asr"), &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("asr http %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	var out ASRResult
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("asr decode: %v", err)
	}
	return &out, nil
}

// ---- Chat / Vision ----

// Message is an OpenAI-compatible chat message. Content may be a string or a []any block array.
type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// ChatResult holds the assistant text plus token usage.
type ChatResult struct {
	Text         string `json:"text"`
	PromptTokens int    `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

// Chat performs a text chat completion.
func (c *Client) Chat(ctx context.Context, msgs []Message) (*ChatResult, error) {
	return c.completion(ctx, msgs)
}

// ImageBlock builds an image_url content block from a local image file (base64 data URL).
func ImageBlock(imgPath string) (map[string]any, error) {
	data, err := os.ReadFile(imgPath)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(imgPath))
	mime := "image/jpeg"
	switch ext {
	case ".png":
		mime = "image/png"
	case ".webp":
		mime = "image/webp"
	case ".gif":
		mime = "image/gif"
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	return map[string]any{
		"type": "image_url",
		"image_url": map[string]any{
			"url": fmt.Sprintf("data:%s;base64,%s", mime, b64),
		},
	}, nil
}

// Vision asks the model about one or more images plus a text prompt.
func (c *Client) Vision(ctx context.Context, prompt string, imagePaths []string) (*ChatResult, error) {
	blocks := []any{map[string]any{"type": "text", "text": prompt}}
	for _, p := range imagePaths {
		b, err := ImageBlock(p)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, b)
	}
	return c.completion(ctx, []Message{{Role: "user", Content: blocks}})
}

func (c *Client) completion(ctx context.Context, msgs []Message) (*ChatResult, error) {
	payload := map[string]any{
		"model":    c.Model,
		"messages": msgs,
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/chat/completions"), bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("chat http %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("chat decode: %v", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("chat: empty choices: %s", truncate(string(body), 200))
	}
	return &ChatResult{
		Text:             parsed.Choices[0].Message.Content,
		PromptTokens:     parsed.Usage.PromptTokens,
		CompletionTokens: parsed.Usage.CompletionTokens,
	}, nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
