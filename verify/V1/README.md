# V1 契约复检（curl）— 使用说明

**状态：已在真实运行的后端上执行，结果 26/26 PASS。见 report.md 与 run-20261118-052158.log。**

## 前置
服务端同时设置 PASSWORD 与 API_TOKEN（覆盖鉴权全分支）：
```
$env:PASSWORD='test123'; $env:API_TOKEN='tok-abc'; $env:PORT='18086'; $env:DATA_DIR='...\_v1data'
cd backend; go build -o server.exe ./cmd/server; .\server.exe
```

## 运行（推荐：PowerShell 5.1 + 原生 curl.exe，已实测）
```
$env:BASE='http://127.0.0.1:18086'
powershell -ExecutionPolicy Bypass -File verify/V1/v1_contract_check.ps1
# 输出重定向/转录保存为 run-<时间>.log 作为证据
```
退出码 = 失败项数量（0=全通过）。

另有 bash 版 `v1_contract_check.sh`（语义等价，供 Linux/git-bash 使用；本机 MSYS 1.0.17 过旧会崩溃，故实跑用 ps1 版）。

## 覆盖项（对应 plan.md §6）
1. /health、/session 免鉴权 → 200
2. 无凭证访问 → 401
3. 错误口令 → 401；4. 正确口令 → 200 + Set-Cookie(session)
5. Cookie 鉴权 → 200
6. X-API-Token 正确 → 200 / 错误(无Cookie) → 401
7. profiles CRUD + api_key 脱敏
8. KB 创建/上传两版/删最新版400/删旧版200/删KB200
9. tasks 创建(multipart)/列表/详情/stages/result 七键/未生成文件404/删除
10. logout

## B3 进度聚合附加核验
见 b3-progress-evidence.md：progress 随已完成阶段按权重上升（实测 10→30），非恒 0 / 非事后 100。

## 产物
- 脚本：v1_contract_check.ps1 / v1_contract_check.sh
- 报告：report.md；B3：b3-progress-evidence.md
- 原始日志：run-20261118-052158.log
