# F1 登录联调 · 证据报告

- 分支：feat/frontend-login（off dev a98f87e）
- 日期：2026-11-18
- 运行环境：真实浏览器（Playwright 1.65.1 + Microsoft Edge，headless）；前端 vite dev http://127.0.0.1:5173（代理 /api → 后端 8080）；后端取 dev 源码构建的二进制，PASSWORD=test123 启动。
- 执行脚本：_fe_evidence_tools/f1_test.cjs（Playwright 驱动真实页面点击 + 网络监听 + 截图）

## 验收项与结果（均 Pass）

| 步骤 | 操作 | 期望 | 实际网络 | 结果 |
|------|------|------|----------|------|
| 1 | 未登录访问 /tasks | 跳 /login | GET /api/v1/session → 200（login_required=true） | ✅ |
| 2 | 输入错误口令点登录 | 提示“密码错误” | POST /api/v1/login → 401 | ✅ |
| 3 | 输入正确口令登录 | 进入 /tasks，下发 Cookie | POST /api/v1/login → 200；GET /session → 200；GET /tasks → 200 | ✅ |
| 4 | 刷新页面 | 保持登录（不被踢回） | GET /session → 200；GET /tasks → 200 | ✅ |
| 5 | 点“退出登录” | 回 /login，Cookie 清除 | POST /api/v1/logout → 200 | ✅ |
| 6 | 退出后再访问 /tasks | 重新跳 /login | GET /session → 200（authenticated=false） | ✅ |

## 关键原始输出（steps.json 摘要）
- 步骤3 登录成功：cookie={name:"session", httpOnly:true, sameSite:"Lax", hasValue:true}；finalUrl=http://127.0.0.1:5173/tasks
- 步骤5 退出：finalUrl=http://127.0.0.1:5173/login；cookieAfter=removed（MaxAge=-1 清除）
- 步骤2：errShown="密码错误"，与后端 login 401 的 {"error":"密码错误"} 对应

## 截图
- 01-未登录访问tasks跳转登录.png
- 02-错误密码提示.png（红字“密码错误”）
- 03-登录成功任务页.png
- 04-刷新后保持登录.png
- 05-退出登录回登录页.png
- 06-退出后再次跳登录.png

## 说明
- Cookie 为 HttpOnly + SameSite=Lax，由后端签名（HMAC），前端不读取其值，仅依赖浏览器自动携带；导出下载等场景由浏览器同源自动带 Cookie（见 F2）。
- console-errors.json 中的 404 为 favicon.ico 缺失（非功能问题）；401 为步骤2 故意用错误口令触发，属预期。
