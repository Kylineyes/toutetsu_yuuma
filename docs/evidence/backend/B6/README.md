# B6 上传安全（大小上限 / 路径校验 / 临时清理）

分支：feat/backend-upload-sec（基于 origin/dev 0a35186）

## 变更
### 1. 上传大小上限
- `config.Config.MaxUploadBytes`（环境变量 `MAX_UPLOAD_BYTES`，单位 MB，默认 2048MB=2GiB）。`getenvInt` 读取。
- `security.go LimitBody(w,r,limit)`：用 `http.MaxBytesReader` 包裹请求体。
- `createTask` 与 `uploadKBVersion` 在 `ParseMultipartForm` 前调用 LimitBody；超限时识别 `http.MaxBytesError` 并返回 **413** 与中文提示「上传内容超过大小上限」。
- `isMaxBytesError` 同时兼容 multipart 的 "request body too large"。

### 2. 路径校验（防目录穿越）
- `security.go SafeJoin(base,name)`：取 `filepath.Base`、拒绝 "."/".."，再用 `filepath.Rel` 二次校验结果仍在 base 内。
- `security.go WithinDir(dir,path)`：目录包含判定（供清理约束）。
- `security.go SanitizeFilename(name)`：剥离目录成分与控制字符，空/点/双点回退 "file"。
- `createTask`：视频文件名用 `SanitizeFilename`；字幕落盘改用 `SafeJoin(uploadDir, kind+"_"+name)`。
- `uploadKBVersion`：上传文件名经 `SanitizeFilename` 后再入 ingest/DB。

### 3. 临时文件清理加固
- `cleanupTaskCache`：待删目录固定为 `TmpDir/intermediates/<filepath.Base(id)>`，并 `WithinDir` 校验严格位于 intermediates 之下，否则 400「非法任务路径」，防止 id 造成穿越误删。

## 边界用例（单元测试，7 用例全绿）
命令：`cd backend && go test ./internal/api/ -v -run "SafeJoin|WithinDir|Sanitize|CopyLimited|LimitBody|IsMaxBytes"`
- TestSafeJoin_Traversal：`../../etc/passwd`、`..\\..\\windows`、`/etc/passwd`、`a/../../b` 均安全（拒绝或落回 base 内）；"."、".." 拒绝。
- TestWithinDir：子路径/自身为真，越界为假。
- TestSanitizeFilename：穿越/反斜杠清洗，空/点回退 file，中文正常保留。
- TestCopyLimited_Oversize：超限返回 ErrUploadTooLarge 且不残留半成品。
- TestCopyLimited_OK：未超限正常写入。
- TestLimitBody_413 / TestIsMaxBytesError_Direct：MaxBytesReader 超限错误被正确识别。
原始输出见 `unittest_output.txt`。

## 端到端边界验证（真实启动，MAX_UPLOAD_BYTES=1MB）
见 `live_boundary_checks.txt`：
1. 2MB 文件上传 → **HTTP 413**，{`error`:`上传内容超过大小上限`}。
2. 文件名 `../../evil.mp4` → 清洗为 `evil.mp4`，落盘 task/<id>/upload/video，data 下 task/ 之外无 evil.mp4（无逃逸）。
3. 预建 tmp/intermediates/<id> 后 DELETE cache → HTTP 200，目录被删除（Test-Path=False）。

## 结论
上传大小上限（413）、非法路径（清洗+SafeJoin+WithinDir 拒绝）、临时清理（路径约束后生效）三项均有边界测试与真实运行证据。
