# V1 契约复检（curl）— 使用说明

**状态：脚本已就绪，实际执行待后端可运行且 lead 通知后再跑（本轮只写脚本，未运行）。**

## 前置
服务端同时设置以下环境变量以覆盖全部鉴权分支：
```
PASSWORD=<口令>
API_TOKEN=<令牌>
```
启动（示例，本地）：
```
cd backend
$env:PASSWORD="changeme"; $env:API_TOKEN="secret"; $env:DATA_DIR="./_v1data"
go run ./cmd/server
```

## 运行（Git Bash / WSL / Linux）
```
BASE=http://127.0.0.1:8080 PASSWORD=changeme API_TOKEN=secret bash verify/V1/v1_contract_check.sh
```
退出码 = 失败项数量（0=全通过）。

## 覆盖项（对应 plan.md §6）
1. 免鉴权 /health、/session → 200
2. 无凭证访问受保护端点 → 401
3. 错误口令登录 → 401
4. 正确口令 → 200 + Set-Cookie(session)
5. Cookie 鉴权访问 → 200
6. X-API-Token 正确 → 200 / 错误 → 401
7. profiles CRUD + api_key 脱敏
8. knowledge-bases 创建 / 上传两版 / 删最新版 400 / 删旧版 200 / 删 KB 200
9. tasks 创建(multipart, video_url) / 列表 / 详情 / stages / result 七键 / 未生成文件 404 / 删除
10. logout

## 产物
- 脚本：`verify/V1/v1_contract_check.sh`
- 运行时请将原始输出重定向保存为 `verify/V1/run-<时间>.log`，作为可复现证据提交。
