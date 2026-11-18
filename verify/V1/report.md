# V1 契约复检报告（真实运行）

- 审计人：Verification Agent（委托子代理 verify），**独立复跑**，未采信 backend 自述。
- 被测基线：dev `0a35186`（B1/B3 已合入），隔离 worktree `<工作目录>\ver_wt`（`git rev-parse HEAD` 复核为 0a35186）。
- 后端：`go build -o server.exe ./cmd/server`（exit 0，17486336 B），以 `PASSWORD=test123 API_TOKEN=tok-abc PORT=18086` 真实启动。
- 脚本：`verify/V1/v1_contract_check.ps1`（PowerShell 5.1 + 原生 curl.exe）。
- 原始输出日志：`verify/V1/run-20261118-052158.log`
- 时间：2026-11-18T05:21:58+08:00
- 命令：`powershell -ExecutionPolicy Bypass -File verify/V1/_v1_curl.ps1`（BASE=http://127.0.0.1:18086）

## 总体结果：**PASS 26 / FAIL 0**（V1_EXIT=0）

| # | 检查项 | 结果 | 关键原始输出 |
|---|--------|------|--------------|
| 1 | GET /health 免鉴权 | PASS | 200 |
| 2 | GET /session 免鉴权 | PASS | 200 |
| 3 | GET /profiles 无凭证 | PASS | 401 |
| 4 | POST /login 错误口令 | PASS | 401 |
| 5 | POST /login 正确口令 | PASS | 200 |
| 6 | Set-Cookie 含 session | PASS | session=... |
| 7 | GET /profiles 带 Cookie | PASS | 200 |
| 8 | X-API-Token 正确 | PASS | 200 |
| 9 | X-API-Token 错误(无Cookie) | PASS | 401 |
| 10 | POST /profiles | PASS | 201 id=1 |
| 11 | api_key 脱敏 | PASS | 响应含 api_key:***，无明文 sk-x |
| 12 | DELETE /profiles/{id} | PASS | 200 |
| 13 | POST /knowledge-bases | PASS | 201 id=1 |
| 14 | POST versions 上传 v1 | PASS | 201 version:1 |
| 15 | POST versions 上传 v2 | PASS | 201 version:2 |
| 16 | DELETE 最新版本 | PASS | 400（保护：不能删最新版） |
| 17 | DELETE 旧版本 | PASS | 200 |
| 18 | DELETE 知识库 | PASS | 200 |
| 19 | POST /tasks multipart 创建 | PASS | 201 id=... |
| 20 | GET /tasks | PASS | 200 |
| 21 | GET /tasks/{id} | PASS | 200 |
| 22 | GET /tasks/{id}/stages | PASS | 200 |
| 23 | GET /tasks/{id}/result 七键 | PASS | task/cost/subtitles/knowledge_bases/stages/summary_md/knowledge_md 齐 |
| 24 | 未生成 summary.md 下载 | PASS | 404 |
| 25 | DELETE /tasks/{id} | PASS | 200 |
| 26 | POST /logout | PASS | 200 |

## 关键原始输出摘录（run-20261118-052158.log）
```
== 4. 正确口令 + Cookie ==
  [PASS] POST /login 正确口令 -> 200
  [PASS] Set-Cookie 含 session
== 8. knowledge-bases 版本与删除保护 ==
  [PASS] 上传 v1 (201)
  [PASS] 上传 v2 (201)
  [PASS] 删最新版 -> 400(保护)
  [PASS] 删旧版 -> 200
  [PASS] 删KB -> 200
== 9. tasks 创建/列表/详情/stages/result ==
  [PASS] result 七键齐
  [PASS] 未生成 summary.md -> 404
  [PASS] DELETE task 200
================ 汇总: PASS=26 FAIL=0 ================
V1_EXIT=0
```

## B3 进度聚合真实运行验证（本轮重点）
详见 `verify/V1/b3-progress-evidence.md`。要点：
- 读码：`ComputeProgress` 按权重 import10/audio10/asr25/frame10/vision15/fuse10/summary20 聚合；`StartStage/UpdateStageProgress/FinishStage` 均调用 `RecomputeTaskProgress` 实时回写。
- 实测（带音轨视频、无 profile）：import+audio+fuse 三阶段 done → 任务 `progress=30`；轮询观测到 progress 由 10 → 30 随阶段完成上升；summary 因未配置 profile 失败，该阶段计 0。
- 结论：**progress 按已完成阶段加权聚合，既非恒 0 也非事后直接 100，B3 在真实运行中成立。**

## 环境限制 / 未覆盖项（如实说明）
- 真实 AI 端到端（ASR/融合/总结/知识点）需真实 provider Key，本轮未覆盖，仅验证到阶段编排与失败落账（summary 因 profile_id=0 失败，expected）。
- ASR/vision/frame 阶段因未配对应 profile 被跳过，未验证真实识别/识图。
- 本机无 Docker，容器内契约复检未做（见 V3 报告）。
- 环境备注：本机 bash 为老旧 MSYS 1.0.17（崩溃），故 V1 采用 PowerShell5.1+curl.exe 方案（脚本内含此说明）。

## 复现命令
```
cd <工作目录>\ver_wt\backend
$env:PASSWORD='test123'; $env:API_TOKEN='tok-abc'; $env:PORT='18086'; $env:DATA_DIR='...\_v1final'
go build -o server.exe ./cmd/server
.\server.exe   # 后台
# 另开：
$env:BASE='http://127.0.0.1:18086'; powershell -ExecutionPolicy Bypass -File ..\verify\V1\v1_contract_check.ps1
```
