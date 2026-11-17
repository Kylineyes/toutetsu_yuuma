# B2 协议实测修正

分支：feat/backend-protocols（基于 origin/dev cea0eeb，含 B5/B6 基线）

## 一、真实端点实测（magpie http://<部署机IP>:30131）
magpie 实际同时支持四种 chat 协议，均已用 backend 内真实 `deepseek.Client` 通过：
| 协议 | 结果 | 说明 |
|---|---|---|
| openai_chat | ✅ `你好` | /chat/completions |
| openai_responses | ✅ `你好` | /responses，优先 output_text |
| anthropic_messages | ✅ `你好` | /v1/messages，x-api-key |
| gemini | ✅ `你好`（修复后） | /v1beta/models/{m}:generateContent |
| ASR（deepseek_asr / 各 whisper） | ❌ 无通道 | /v1/tools/asr 404；whisper 模型 404 |

原始日志见 `real_endpoint_probe.txt`。

## 二、发现并修复的解析缺陷（gemini thought 片段）
实测 gemini 推理模型偶尔在 `candidates[0].content.parts` 里返回 **`thought: true` 的思考片段**，
原实现会把所有 parts 的 text 拼接 → 输出混入英文思考内容（实测曾出现：
`[The user asks me... 你好]`）。真实原始响应示例（未修复前）：
```json
{"candidates":[{"content":{"parts":[{"text":"The user is asking...","thought":true},{"text":"你好"}]}}]}
```
**修复**（client.go chatGemini）：parts 结构新增 `Thought bool`，拼接时 `if pt.Thought { continue }` 跳过思考片段，只输出正式回复。
修复后真实端点连测 5 次均为干净的 `你好`。

## 三、可复现 mock 测试（httptest，打各协议解析路径）
命令：`cd backend && go test ./internal/deepseek/ -v -run TestProtocol`
mock server 模拟各协议真实响应体，校验请求结构 + 解析：
- TestProtocol_OpenAIChat：model/Authorization、choices[].message.content、usage
- TestProtocol_OpenAIResponses：output_text 优先 + output[].content[].text 回退拼接
- TestProtocol_Anthropic：content[].text 拼接、x-api-key、max_tokens
- TestProtocol_Gemini：candidates[0].content.parts[].text 拼接、usageMetadata
- TestProtocol_GeminiSkipsThought（回归）：thought=true 片段被跳过
- TestProtocol_ASRDeepSeek / ASROpenAI：/v1/tools/asr、/v1/audio/transcriptions 解析 text
- TestProtocol_HTTPError：非 200 返回错误
结果：**8/8 PASS**。原始输出见 `mock_unittest_output.txt`。

## 结论
四种 chat 协议真实可用并通过实测；ASR 无通道（环境限制，与 B1 一致）；修正了 gemini 推理片段混入输出的解析缺陷，并补回归测试。
