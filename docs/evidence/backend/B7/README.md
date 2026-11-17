# B7 空列表契约：返回 [] 而非 null

分支：feat/backend-emptylist（基于 origin/dev 952fc86）

## 问题
`listKBs` / `listProfiles`（以及 `listTasks` 等）在底层返回 nil slice 时，`json.Encode` 编码为 `null`，
前端对其 `.length` 会崩溃（前端 F6 已兜底，但后端也应契约正确）。

## 方案（统一在 writeJSON 层处理，最通用）
- 在 `api.WriteJSON` 编码前调用 `normalizeNilSlice(v)`：
  - 仅当 v 的 reflect.Kind()==Slice 且 IsNil() 时，用 `reflect.MakeSlice(type,0,0)` 替换为同类型空切片 → 编码为 `[]`。
  - 非切片类型（map/struct/指针/接口）语义完全不变（nil map 仍为 null，不做 {} 变更，避免改变其它语义）。
- 一处修改覆盖所有列表接口（knowledge-bases / profiles / tasks 等），无需逐个 handler 判空。

## 单元测试（5 用例全绿）
命令：`cd backend && go test ./internal/api/ -v -run "EmptyIsArray|NormalizeNilSlice|WriteJSON_NilSlice"`
- TestListKBs_EmptyIsArray：空库 listKBs → body 以 '[' 开头、反序列化为非 nil 空切片。
- TestListProfiles_EmptyIsArray：空库 listProfiles → []。
- TestListTasks_EmptyIsArray：空库 listTasks → []（通用兜底）。
- TestNormalizeNilSlice：nil []int→非 nil 空切片；非 nil 切片原样；nil 接口/map/普通值语义不变。
- TestWriteJSON_NilSliceIsArray：writeJSON 编码 nil []db.KnowledgeBase → []。
原始输出见 `unittest_output.txt`。

## 真实运行验证（空库 PORT=18087）
见 `live_empty_lists.txt`：
```
GET /api/v1/knowledge-bases → []
GET /api/v1/profiles        → []
GET /api/v1/tasks           → []
```

## 结论
空集统一返回 []，前端 .length 不再崩溃；writeJSON 层通用处理，其它 nil 语义未改动；单测+真实运行双证据。
