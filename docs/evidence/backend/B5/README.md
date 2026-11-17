# B5 单元测试（subtitle / kb / auth）

分支：feat/backend-tests（基于 origin/dev 0a35186）

## 新增测试
### subtitle（backend/internal/subtitle/subtitle_test.go，12 用例）
- 解析：TestParseSRT（时间戳换算+去标签+多行）、TestParseVTT、TestParseASS（Format 定位 Text 列、\N 换行）、TestParseTXT、TestParseDanmuXML
- 格式化：TestFormatSRTRoundtrip（FormatSRT→parseSRT 回读）
- 融合：TestFusePriorityNativeWins（native 优先去重 asr）、TestFuseASRGapFill（asr 补空缺）、TestFuseVisionAppended（vision 追加+按时间排序）、TestFuseDedupe（文本去重）
- 术语纠错：TestApplyTerms（替换且不改原切片）、TestApplyTermsEmpty

### kb（backend/internal/pipeline/kb_test.go，4 用例）
- TestIngestAndLoadCSV：内容寻址写入 + CSV 术语加载 + term_count
- TestIngestJSON：JSON 术语解析
- TestContentAddressingDedupe：相同内容只计一次 newBytes、sha 相同
- TestDeleteVersionAndGC：删除版本后无引用 blob 回收、被其它版本引用的 blob 保留

### auth（backend/internal/api/auth_test.go，8 用例）
- TestExemptPath / TestAuth_ExemptPathsBypass：免鉴权路径
- TestSessionValueHMAC：session 签名为 64 hex、同 secret 稳定、异 secret 不同
- TestAuth_TokenHeader：X-API-Token 正确放行/错误 401
- TestAuth_BearerAndQuery：Authorization: Bearer 与 ?token= 两种携带
- TestAuth_SessionCookie：正确 cookie 放行/伪造或缺失 401
- TestAuth_Open：token、password 皆空开放
- TestAuth_NonAPI：非 /api/ 前缀直接放行

## 验证命令与结果
```
cd backend
go build ./...      # BUILD 0
go vet ./...        # VET 0
go test ./... -v -count=1
```
结果：**4 个包全部 ok，32 个用例全部 PASS，0 FAIL**（internal/db 为既有 B3 测试一并跑绿）。
完整原始输出见 `go_test_all.txt`。

## 说明
- 现存 internal/config/media/deepseek/worker 仍为 no test files（本轮未纳入 B5 范围，B5 只覆盖 subtitle/kb/auth 三块）。
- 所有测试均为纯单元测试，无外部依赖（不触网、不依赖 ffmpeg）。
