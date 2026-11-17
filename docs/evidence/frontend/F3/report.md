# F3 配置方案 · 证据报告

- 分支：feat/frontend-profiles（off feat/frontend-login 7ced3e0）
- 环境：真实浏览器 Playwright+Edge；前端 vite dev 5173（fe_wt）；后端 dev 源码二进制 PASSWORD=test123。
- 变更（frontend/src/views/ProfilesView.vue）：
  1) ASR 协议补全 openai_whisper（后端 asrProtocols 含之）；
  2) 新增 validate()：温度 0~2（空=默认）、超时为正整数秒、extra 必须为合法 JSON 对象；
  3) 列表回显新增“extra”列，超时显示带 s 单位。

## 验收项与结果（全部 Pass，真实浏览器）

| 步骤 | 操作 | 期望 | 实测 | 结果 |
|------|------|------|------|------|
| 1 | 能力 总结→ASR | 协议选项随之切换 | 总结:OpenAI Chat/Responses/Anthropic/Gemini；ASR:DeepSeek ASR/OpenAI Audio/Whisper；默认协议 deepseek_asr | ✅ |
| 2 | 温度填 5 提交 | 拒绝并提示 | 提示“温度须为 0~2 之间的数字，或留空使用默认” | ✅ |
| 3 | 超时填 0 提交 | 拒绝并提示 | 提示“超时须为正整数秒” | ✅ |
| 4 | extra 填 {bad json 提交 | 拒绝并提示 | 提示“extra 不是合法 JSON: Expected property name...” | ✅ |
| 5 | 合法提交 temp=0.7/timeout=99/extra={"top_p":0.9} | 201 且列表回显 | POST /profiles→201；列表行 Temp=0.7 超时=99s extra={"top_p":0.9} | ✅ |

## 关键原始输出（steps.json）
见 steps.json，pass 全 true。步骤5 createReq=[{m:POST,u:/api/v1/profiles,s:201}]。

## 截图
- 01-能力为asr时协议选项.png（能力=ASR，协议下拉为 ASR 三项）
- 02-温度越界校验.png / 03-超时校验.png / 04-extra非法校验.png
- 05-合法创建与回显.png（列表 Temp 0.7/超时 99s/extra {"top_p":0.9}）

## 说明 / 风险
- 截图因多次重跑积累多条 F3-echo 测试方案，属测试数据，非功能问题。
- 后端对 api_key 返回 ***，不影响校验/回显。
