# v1 限制

- 只处理视频源文件 / 直链，不做平台页解析（yt-dlp 后续）
- DeepSeek ASR 无词级时间戳 → ASR 字幕为分片级近似；fused.srt 以原字幕时间轴为准
- 截图仅 simple（固定间隔）；paused 状态预留未实现
- 成本仅记录展示不限制；单租户

> 返回上级：[README](../README.md)
