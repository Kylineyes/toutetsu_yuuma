# 配置与鉴权

## 环境变量（后端）

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| HOST | 0.0.0.0 | 绑定地址（单地址）|
| PORT | 8080 | 监听端口 |
| DATA_DIR | ./data | 数据根目录 |
| TMP_DIR | 系统临时目录 | 中间产物根（${TMP_DIR}/intermediates/{task}）|
| PASSWORD | 空 | Web UI 登录口令；空=不启用登录 |
| API_TOKEN | 空 | 第三方令牌（X-API-Token / Bearer / ?token=）；空=不校验 |
| SESSION_SECRET | 取 PASSWORD | 登录 Cookie 签名密钥 |
| MAX_GO_CONCURRENCY | 4 | 全局并发上限（ASR 分片 / 识图帧）|
| DEFAULT_SCREENSHOT_INTERVAL | 60 | 默认截图间隔（秒）|
| MAX_FRAMES | 120 | 单任务最多截帧数 |
| TZ | Asia/Shanghai | 时区 |

## 鉴权

- **Web UI**：前端登录页输入 `PASSWORD` → 服务端下发签名 Cookie（HMAC）→ 后续请求凭 Cookie。
- **第三方**：请求头 `X-API-Token: <API_TOKEN>`（或 `Authorization: Bearer`；下载可 `?token=`）。
- 免鉴权：`/health`、`/login`、`/session`。两条通道都空 = 完全开放。

## 配置方案（Profile）

每类能力一份：`capability` = summary/asr/vision/reading；建任务时每类各选一个。

- 字段：`protocol / base_url / api_key / model / temperature / timeout_seconds / extra(JSON)`
- chat 协议（summary/vision/reading）：`openai_chat / openai_responses / anthropic_messages / gemini`
- audio 协议（asr）：`deepseek_asr`(/v1/tools/asr) / `openai_audio`(/v1/audio/transcriptions)

> 返回上级：[README](../README.md)
