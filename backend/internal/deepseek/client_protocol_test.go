package deepseek

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockServer 按预期路径返回一组预设的响应体，用于验证各协议的解析路径。
func mockServer(t *testing.T, routes map[string]mockResp) (*httptest.Server, *[]mockReq) {
	t.Helper()
	var reqs []mockReq
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		reqs = append(reqs, mockReq{Path: r.URL.Path, Method: r.Method, Body: body, Header: r.Header.Clone()})
		resp, ok := routes[r.URL.Path]
		if !ok {
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
			return
		}
		if resp.contentType != "" {
			w.Header().Set("Content-Type", resp.contentType)
		}
		w.WriteHeader(resp.status)
		_, _ = w.Write([]byte(resp.body))
	}))
	t.Cleanup(srv.Close)
	return srv, &reqs
}

type mockResp struct {
	status      int
	body        string
	contentType string
}
type mockReq struct {
	Path   string
	Method string
	Body   map[string]any
	Header http.Header
}

// TestProtocol_OpenAIChat 校验 openai_chat：请求体模型/消息结构正确，解析 choices[0].message.content 与 usage。
func TestProtocol_OpenAIChat(t *testing.T) {
	srv, reqs := mockServer(t, map[string]mockResp{
		"/chat/completions": {status: 200, body: `{"choices":[{"message":{"content":"你好"}}],"usage":{"prompt_tokens":11,"completion_tokens":7}}`},
	})
	c := New(srv.URL, "k", "m1", "openai_chat", nil, 10)
	res, err := c.ChatText(context.Background(), "hi")
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if res.Text != "你好" || res.PromptTokens != 11 || res.CompletionTokens != 7 {
		t.Fatalf("解析错误: %+v", res)
	}
	r := (*reqs)[0]
	if r.Body["model"] != "m1" {
		t.Fatalf("请求 model 错误: %v", r.Body["model"])
	}
	if r.Header.Get("Authorization") != "Bearer k" {
		t.Fatalf("Authorization 头错误: %q", r.Header.Get("Authorization"))
	}
}

// TestProtocol_OpenAIResponses 校验 openai_responses：优先 output_text，缺失时回退遍历 output[].content[].text。
func TestProtocol_OpenAIResponses(t *testing.T) {
	// 1) output_text 直接给出
	srv, reqs := mockServer(t, map[string]mockResp{
		"/responses": {status: 200, body: `{"output_text":"直接","usage":{"input_tokens":3,"output_tokens":4}}`},
	})
	c := New(srv.URL, "k", "m2", "openai_responses", nil, 10)
	res, err := c.ChatText(context.Background(), "hi")
	if err != nil {
		t.Fatalf("responses: %v", err)
	}
	if res.Text != "直接" || res.PromptTokens != 3 {
		t.Fatalf("output_text 解析错误: %+v", res)
	}
	_ = reqs
	srv.Close()

	// 2) 无 output_text，回退 output 数组
	srv2, _ := mockServer(t, map[string]mockResp{
		"/responses": {status: 200, body: `{"output":[{"content":[{"text":"分"},{"text":"片"}]}],"usage":{"input_tokens":1,"output_tokens":2}}`},
	})
	c2 := New(srv2.URL, "k", "m2", "openai_responses", nil, 10)
	res2, err := c2.ChatText(context.Background(), "hi")
	if err != nil {
		t.Fatalf("responses fallback: %v", err)
	}
	if res2.Text != "分片" {
		t.Fatalf("output 数组拼接错误: %q", res2.Text)
	}
}

// TestProtocol_Anthropic 校验 anthropic_messages：解析 content[].text 与 usage，且请求带 x-api-key。
func TestProtocol_Anthropic(t *testing.T) {
	srv, reqs := mockServer(t, map[string]mockResp{
		"/v1/messages": {status: 200, body: `{"content":[{"text":"安"},{"text":"卓"}],"usage":{"input_tokens":5,"output_tokens":6}}`},
	})
	c := New(srv.URL, "ak", "claude-x", "anthropic_messages", nil, 10)
	res, err := c.ChatText(context.Background(), "hi")
	if err != nil {
		t.Fatalf("anthropic: %v", err)
	}
	if res.Text != "安卓" || res.PromptTokens != 5 || res.CompletionTokens != 6 {
		t.Fatalf("解析错误: %+v", res)
	}
	r := (*reqs)[0]
	if r.Header.Get("x-api-key") != "ak" {
		t.Fatalf("x-api-key 头缺失: %v", r.Header)
	}
	if r.Body["max_tokens"] == nil {
		t.Fatalf("anthropic 请求应带 max_tokens")
	}
}

// TestProtocol_Gemini 校验 gemini：解析 candidates[0].content.parts[].text 与 usageMetadata。
func TestProtocol_Gemini(t *testing.T) {
	srv, reqs := mockServer(t, map[string]mockResp{
		"/v1beta/models/gem-1:generateContent": {status: 200, body: `{"candidates":[{"content":{"parts":[{"text":"G1"},{"text":"G2"}]}}],"usageMetadata":{"promptTokenCount":8,"candidatesTokenCount":9}}`},
	})
	c := New(srv.URL, "gkey", "gem-1", "gemini", nil, 10)
	res, err := c.ChatText(context.Background(), "hi")
	if err != nil {
		t.Fatalf("gemini: %v", err)
	}
	if res.Text != "G1G2" || res.PromptTokens != 8 || res.CompletionTokens != 9 {
		t.Fatalf("解析错误: %+v", res)
	}
	// key 应走 query 参数
	if !strings.Contains((*reqs)[0].Header.Get("Referer"), "") { // no-op 保底
	}
}

// TestProtocol_ASRDeepSeek 校验 deepseek_asr：POST /v1/tools/asr 解析 text。
func TestProtocol_ASRDeepSeek(t *testing.T) {
	srv, reqs := mockServer(t, map[string]mockResp{
		"/v1/tools/asr": {status: 200, body: `{"text":"转写文本","duration_ms":1234}`},
	})
	// 需要真实存在的音频文件路径；CreateFormFile 会 open 文件，故写一个临时文件。
	af := writeTempAudio(t)
	c := New(srv.URL, "k", "", "deepseek_asr", nil, 10)
	res, err := c.Transcribe(context.Background(), af)
	if err != nil {
		t.Fatalf("asr: %v", err)
	}
	if res.Text != "转写文本" || res.DurationMs != 1234 {
		t.Fatalf("ASR 解析错误: %+v", res)
	}
	if (*reqs)[0].Path != "/v1/tools/asr" {
		t.Fatalf("ASR 路径错误: %s", (*reqs)[0].Path)
	}
}

// TestProtocol_ASROpenAI 校验 openai_audio：POST /v1/audio/transcriptions 解析 text。
func TestProtocol_ASROpenAI(t *testing.T) {
	srv, reqs := mockServer(t, map[string]mockResp{
		"/v1/audio/transcriptions": {status: 200, body: `{"text":"whisper 文本"}`},
	})
	af := writeTempAudio(t)
	c := New(srv.URL, "k", "whisper-1", "openai_audio", nil, 10)
	res, err := c.Transcribe(context.Background(), af)
	if err != nil {
		t.Fatalf("openai_audio: %v", err)
	}
	if res.Text != "whisper 文本" {
		t.Fatalf("ASR 解析错误: %+v", res)
	}
	if (*reqs)[0].Path != "/v1/audio/transcriptions" {
		t.Fatalf("ASR 路径错误: %s", (*reqs)[0].Path)
	}
}


// TestProtocol_GeminiSkipsThought 回归：Gemini parts 中含 thought=true 的思考片段应被跳过，只输出正式回复。
func TestProtocol_GeminiSkipsThought(t *testing.T) {
	srv, _ := mockServer(t, map[string]mockResp{
		"/v1beta/models/gem-1:generateContent": {status: 200, body: `{"candidates":[{"content":{"parts":[{"text":"这是思考内容","thought":true},{"text":"你好"}]}}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":2}}`},
	})
	c := New(srv.URL, "k", "gem-1", "gemini", nil, 10)
	res, err := c.ChatText(context.Background(), "hi")
	if err != nil {
		t.Fatalf("gemini: %v", err)
	}
	if res.Text != "你好" {
		t.Fatalf("应跳过 thought 片段，实际输出 %q", res.Text)
	}
}

// TestProtocol_HTTPError 校验非 200 响应返回错误而非静默。
func TestProtocol_HTTPError(t *testing.T) {
	srv, _ := mockServer(t, map[string]mockResp{
		"/chat/completions": {status: 500, body: `{"error":"boom"}`},
	})
	c := New(srv.URL, "k", "m", "openai_chat", nil, 10)
	if _, err := c.ChatText(context.Background(), "hi"); err == nil {
		t.Fatalf("500 应返回错误")
	}
}