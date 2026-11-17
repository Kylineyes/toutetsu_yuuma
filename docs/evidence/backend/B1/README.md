# B1 真实 AI 端到端（含 ASR 降级说明）

> **重要：ASR 为降级路径。** 本环境无真实可用 ASR 通道，`fused.srt` 的输入为本地生成的 SRT（内容与视频主题一致），而非真实语音转录。详见下文实测证据。

## 一、环境结论（实测）
| 目标 | 结果 | 证据 |
|---|---|---|
| openai_chat 对话（magpie 网关） | ✅ 可用 HTTP 200，返回中文正常 | 账号 workbuddy/deepseek-v4.1-flash |
| magpie `/v1/tools/asr` | ❌ HTTP 404（magpie 未提供该工具端点） | 见 `asr_probe_error.txt` |
| magpie `/v1/audio/transcriptions` 各 whisper 模型 | ❌ 全部 404（WorkBuddy 通道不提供语音模型） | 探测日志 |
| 103.85.25.158:8080 | 实为 API 网关 Web UI（返回 HTML） | 探测日志 |
| DEEPSEEK_API_KEY 直连 api.deepseek.com | ❌ 401 invalid（非官方 key） | 探测日志 |

以 **backend 内真实客户端** `deepseek.Client.Transcribe`（protocol=deepseek_asr）直连 magpie 的原始错误：
```
ASR ERROR: asr http 404: {"error":{"message":"magpie serves /v1/chat/completions, /v1/responses, ... /v1/audio/speech and /v1beta/models/*", ...}}
```
（完整见 `asr_probe_error.txt`）

## 二、本次 e2e 实际执行的任务
- 任务 ID：`1794949904744510100-49904`
- 输入：1 分钟测试视频（`video.mp4`，60s，640x360）+ 本地 SRT（`native.srt`，12 条，主题=光的折射/斯涅尔定律/全反射，与视频主题一致）
- 方案：summary_profile = magpie openai_chat（workbuddy/deepseek-v4.1-flash，temperature=0.3）；reading_depth=1
- 命令：
  ```
  go build -o server.exe ./cmd/server
  DATA_DIR=.../data TMP_DIR=.../tmp PORT=18081 ./server.exe
  curl -s -X POST -F video_file=@video.mp4 -F subtitle_native=@native.srt -F summary_profile_id=1 -F reading_depth=1 -F screenshot_interval=10 http://127.0.0.1:18081/api/v1/tasks
  ```

### task_stages（真实执行，含耗时；JSON 见 `stages.json`）
| stage | status | duration_ms |
|---|---|---|
| import | done | 783 |
| audio | done | 1105 |
| fuse | done | 180 |
| summary | done | 25316 |

> 说明：此任务用 native 字幕，`asr` 阶段按设计不触发（无 asr 行）。`vision` 因未选 vision_profile 且未开 generate_vision 而未触发；summary 因 reading_depth=1 但无 frames（未选 vision/reading profile → 不抽帧）跳过回看轮。故实际出现 import/audio/fuse/summary 四阶段，全部 done。**如实记录，未强凑。**

## 三、DoD 产物（均非空，真实模型生成）
```
- output/summary.md     2028 字节  含：概述 + 4 个主题分节（主题一~四）
- output/knowledge.md   4484 字节  8 张知识点卡片 + 目录（标题/一句话结论/详细说明/标签/时间戳）
- subs/fused.srt        1504 字节  48 行，12 条字幕
```
- 生产成本：prompt_tokens=543，completion_tokens=5345（真实 LLM 调用）
- 下载接口验证：`GET /tasks/{id}/files/summary.md|knowledge.md|fused.srt` 均 200（本目录同名文件即下载所得）

## 四、ASR 缺失的反证（强制走 ASR 的任务）
- 任务 ID：`1794949955537749100-49955`（asr_profile_id=2 protocol=deepseek_asr，generate_ai_subtitle_via_asr=1，无 native 字幕）
- 结果：asr 阶段 running 8s 后无任何 cue 产出（`subs/asr.srt` 不存在，`fused.srt` 为 0 字节），summary 阶段基于空字幕，模型**拒绝编造**，输出"未收到字幕正文，无法可靠概括"（见 `asr_forced_result.json`）。
- stages 见 `asr_forced_stages.json`。此任务证明：**真实 ASR 不可用，若强制走 ASR 则无法得到有效字幕**。

## 五、复现与遗留风险
- 本机已有真实可用 openai_chat（magpie）；ASR 缺失属**环境硬限制**，非代码缺陷，也无可用 key/端点可下发。
- 若后续提供可用 ASR base_url+key+协议（如真实 deepseek ASR 或 openai/whisper），把方案 protocol 设为 deepseek_asr/openai_audio 即可切回真实 ASR；pipeline 的 ASR→fuse→summary 逻辑本身已由本任务的下游验证。
- 证据文件：task.json / task_result.json / summary.md / knowledge.md / fused.srt / stages.json / asr_probe_error.txt / asr_forced_result.json / asr_forced_stages.json
