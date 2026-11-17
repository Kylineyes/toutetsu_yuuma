# M3 本地端到端冒烟（非容器，替代 docker 容器内验证）

- 执行人：Verification Agent（委托子代理 verify），独立复跑。
- 基线：dev `952fc86`（worktree ver_wt）。
- 目的：在**无 Docker** 前提下，用「后端二进制 + 前端 dist 静态托管反代 /api」复现 `docker-compose.yml` 描述的架构，证明其可跑通（仅缺容器封装）。
- 环境：Windows；Go 1.27.4 编译后端；Node 26 + vite build 前端 dist；Edge 157(CDP) 真实浏览器操作。

## 一、部署拓扑（对齐 compose）
| compose | 本地等价 | 状态 |
|---------|----------|------|
| backend 容器 8080 | `backend/server.exe` PORT=18090，PASSWORD=test123，DATA_DIR=_m3data，TMP_DIR | 起 |
| frontend 容器 nginx 80→8088 | Node 静态服务器 18091 托管 frontend/dist，`/api` 反代→18090（等价 nginx proxy_pass） | 起 |
| 卷 ./data:/data | DATA_DIR=_m3data（宿主目录持久化） | 等价 |
| 依赖 ffmpeg | 本机 ffmpeg 可用（audio 阶段真实执行成功，见下） | 等价 |

## 二、端到端流程与结果：**PASS 8 / 8**
真实浏览器：登录 → 新建任务(上传带音轨视频,不选 profile) → 详情观察阶段 → result → 登录态下载。

| 步骤 | 结果 | 证据 |
|------|------|------|
| 登录成功跳转 /tasks | PASS | M3-1-login.png |
| 新建任务填写 | PASS | M3-2-create-form.png |
| 提交任务 POST /tasks → **201** | PASS | M3-3-detail-created.png |
| 跳转详情页 | PASS | M3-3-detail-created.png |
| **import 阶段真实执行** | PASS（done 100%, 0.7s） | M3-4-detail-stages-final.png |
| **audio 阶段真实执行（ffmpeg）** | PASS（done 100%, 0.9s） | M3-4-detail-stages-final.png |
| 阶段表有记录（import/audio/fuse done，summary failed） | PASS | M3-4-detail-stages-final.png |
| result 接口返回 task/stages 等 | PASS | — |
| 登录态下载鉴权通过（fused.srt=200 其余404，均非401） | PASS | 见下 |

阶段实测（详情页截图 M3-4）：总进度 **30%**（import10+audio10+fuse10 加权，符合 B3）；import/audio/fuse=**done 100% 耗时 0.7~0.9s**（真实 ffmpeg）；summary=failed（未选 profile，预期）。
下载实测：`{"summary.md":404,"knowledge.md":404,"fused.srt":200,"asr.srt":404,"vision.srt":404}` —— **fused.srt=200 为真实生成并通过登录 Cookie 下载成功**；其余未生成故 404；均非 401（鉴权通过）。

**结论：compose 描述的「nginx 反代 /api → Go 后端 + SQLite + ffmpeg」架构在当前实现下可跑通，仅缺容器打包（本机无 Docker 未做镜像构建）。**

## 三、过程中发现的问题（需 frontend 修复）—— **F5 空知识库列表导致新建任务页崩溃**
- 现象：**全新部署（DB 无任何知识库）** 时，进入「新建任务」页，内容区**完全空白**（仅剩侧栏），所有表单/文件插槽不渲染。
- 复现（独立、真实）：
  1. 后端全新 DATA_DIR 启动；`GET /api/v1/knowledge-bases` 返回 JSON `null`（非 `[]`）。
  2. 浏览器打开 /tasks/new → 控制台报 `TypeError: Cannot read properties of null (reading 'length')`，main 内容长度=7（空）。
  3. 一旦创建任意 1 个知识库（返回数组），/tasks/new 恢复正常（5 个 file 输入、h2=新建任务）。
- 证据截图：`M3-emptyKB-crash-tasks-new.png`（内容区空白）。
- 根因：frontend TaskCreate.vue `kbs.value = await api.kbs()`；当后端返回 `null` 时 `kbs.value=null`，模板 `v-if="!kbs.length"` 访问 `null.length` 抛错。
- 修复建议（任选其一，推荐双保险）：
  - 前端：`kbs.value = (await api.kbs()) || []`（profile 同理 `|| []`）；
  - 后端：ListKnowledgeBases/ListProfiles 空集返回 `[]` 而非 `null`（Go 里初始化为 `make([]T,0)`）。
- 影响：M3 容器部署若首次启动（空库）直接进新建任务页会白屏；属**新发现缺陷**，V2 当时因已预置知识库未暴露。

## 四、环境缺口（如实）
- 本机无 docker/docker-compose/podman/nerdctl/buildah，无 docker 服务；`docker compose build` 与容器内端到端**未验证**。本地端到端已尽可能等价复现（见上）。

## 五、证据清单（verify/M3/）
M3-1-login.png、M3-2-create-form.png、M3-3-detail-created.png、M3-4-detail-stages-final.png、M3-emptyKB-crash-tasks-new.png

## 六、复现命令（本地等价）
```
cd backend; go build -o server.exe ./cmd/server
$env:PASSWORD='test123'; $env:PORT='18090'; $env:DATA_DIR='..\_m3data'; .\server.exe   # 后台
node _serve.js frontend/dist http://127.0.0.1:18090 18091                              # 静态+/api反代
# Edge --headless --remote-debugging-port=9222 + CDP 脚本：登录→/tasks/new 上传视频→提交→/tasks/{id} 观察 stages→下载
```
