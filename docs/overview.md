# 项目概览

> 项目名 **Toutetsu Yuuma** 取自「**饕餮尤魔**」的日语读音（とうてつ ゆうま）。

`toutetsu_yuuma` 是一个**长期运行、可 Docker 部署**的「视频阅读」工程：融合多种字幕来源理解视频内容，产出**视频总结**与**知识点**，并支持用领域**知识库**做专有名词纠错。

## 技术栈

- 前端：Vue 3 + Vite + vue-router（构建后 nginx 托管）
- 后端：Go（net/http），内置 ffmpeg
- 数据库：SQLite
- AI：按「配置方案」调用（支持 OpenAI Chat/Responses、Anthropic Messages、Gemini、DeepSeek ASR、OpenAI Audio 等协议）

## 功能

- 任务 = 一个视频；异步流水线、**阶段可并行**、断点续跑、分阶段重试，阶段进度与耗时单独记录
- 多字幕来源融合：原生 / AI / 识图 / 弹幕（+ 服务可用 ASR、识图模型自行生成）
- 服务端完成：ffmpeg 抽音轨（16k 单声道 WAV）、VAD 静音切分（≤120s 逐段 ASR）、按间隔截帧
- 产出 `summary.md`（分析时间+概述+按主题分节的具体内容）+ `knowledge.md`（知识点卡片）
- 阅读理解深度 T：总结阶段可多轮「回看画面」（视觉 AI 阅读理解）
- 知识库：结构化术语词典、版本化、内容寻址去重（`blobs/` + `manifest.json`）、可多选、冲突按勾选顺序

> 下一阶段设计：单任务处理计划（来源台账 / 质检融合 / 术语纠错 / 用语库 / 总结）见 [plans/README.md](plans/README.md)。

> 返回上级：[README](../README.md)
