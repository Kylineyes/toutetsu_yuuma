#!/usr/bin/env bash
# V1 契约复检脚本（curl）—— 待后端可运行且 lead 通知后执行
# 用法：
#   BASE=http://127.0.0.1:8080 PASSWORD=changeme API_TOKEN=secret ./v1_contract_check.sh
# 说明：脚本只读+会创建/删除少量测试数据；需服务端同时设置 PASSWORD 与 API_TOKEN 以覆盖 401 分支。
set -uo pipefail

BASE="${BASE:-http://127.0.0.1:8080}"
API="$BASE/api/v1"
PASSWORD="${PASSWORD:-changeme}"
API_TOKEN="${API_TOKEN:-}"
VIDEO_URL="${VIDEO_URL:-https://example.com/test.mp4}"   # 仅用于创建任务(异步)，不实际下载即可验证落库/契约
COOKIEJAR="$(mktemp)"
PASS=0; FAIL=0
pass(){ echo "  [PASS] $1"; PASS=$((PASS+1)); }
fail(){ echo "  [FAIL] $1  -> $2"; FAIL=$((FAIL+1)); }
code(){ curl -s -o /dev/null -w '%{http_code}' "$@"; }
json(){ curl -s "$@"; }

echo "== 目标: $BASE =="
echo "== 1. 免鉴权端点 =="
[ "$(code $API/health)" = "200" ] && pass "GET /health 200" || fail "GET /health" "非200"
[ "$(code $API/session)" = "200" ] && pass "GET /session 200" || fail "GET /session" "非200"

echo "== 2. 401 未授权（无 cookie/token，且已设定口令时） =="
c=$(code "$API/profiles")
if [ "$c" = "401" ]; then pass "GET /profiles 无凭证 -> 401"; else fail "GET /profiles 无凭证" "期望401实得$c（若服务端未设PASSWORD/API_TOKEN则本项跳过=开放模式）"; fi

echo "== 3. 登录错误密码 -> 401 =="
c=$(code -X POST "$API/login" -H 'Content-Type: application/json' -d '{"password":"__wrong__"}')
[ "$c" = "401" ] && pass "POST /login 错误口令 -> 401" || fail "POST /login 错误口令" "实得$c"

echo "== 4. 登录正确密码 -> 200 并种 Cookie =="
c=$(code -c "$COOKIEJAR" -X POST "$API/login" -H 'Content-Type: application/json' -d "{\"password\":\"$PASSWORD\"}")
[ "$c" = "200" ] && pass "POST /login 正确口令 -> 200" || fail "POST /login 正确口令" "实得$c"
grep -qi 'session' "$COOKIEJAR" && pass "Cookie jar 含 session" || fail "Cookie 检查" "无 session"

echo "== 5. Cookie 鉴权访问受保护端点 =="
c=$(code -b "$COOKIEJAR" "$API/profiles")
[ "$c" = "200" ] && pass "GET /profiles 带Cookie -> 200" || fail "GET /profiles 带Cookie" "实得$c"

echo "== 6. X-API-Token 鉴权（若配置） =="
if [ -n "$API_TOKEN" ]; then
  c=$(code -H "X-API-Token: $API_TOKEN" "$API/profiles")
  [ "$c" = "200" ] && pass "GET /profiles X-API-Token -> 200" || fail "GET /profiles X-API-Token" "实得$c"
  c=$(code -H "X-API-Token: wrong" -H "Cookie:" "$API/profiles")
  [ "$c" = "401" ] && pass "GET /profiles 错误Token -> 401" || fail "错误Token" "实得$c"
else
  echo "  [SKIP] 未提供 API_TOKEN，跳过 token 分支"
fi

echo "== 7. profiles CRUD（Cookie 下）=="
body='{"capability":"summary","protocol":"openai_chat","name":"v1-test","base_url":"https://api.deepseek.com","model":"deepseek-chat","api_key":"sk-x","temperature":0.3,"timeout_seconds":120,"extra":"{}"}'
pid=$(json -b "$COOKIEJAR" -X POST "$API/profiles" -H 'Content-Type: application/json' -d "$body" | sed -n 's/.*"id":\([0-9]*\).*/\1/p')
if [ -n "$pid" ] && [ "$pid" -gt 0 ] 2>/dev/null; then pass "POST /profiles -> id=$pid"; else fail "POST /profiles" "未返回 id"; pid=0; fi
if [ "$pid" -gt 0 ]; then
  resp=$(json -b "$COOKIEJAR" "$API/profiles/$pid")
  echo "$resp" | grep -q '"api_key":"\*\*\*"' && pass "GET /profile api_key 脱敏 ***" || fail "api_key 脱敏" "$resp"
  c=$(code -b "$COOKIEJAR" -X DELETE "$API/profiles/$pid")
  [ "$c" = "200" ] && pass "DELETE /profiles/$pid -> 200" || fail "DELETE profile" "实得$c"
fi

echo "== 8. knowledge-bases 创建/版本/删除保护 =="
kid=$(json -b "$COOKIEJAR" -X POST "$API/knowledge-bases" -H 'Content-Type: application/json' -d '{"name":"v1-kb-'$RANDOM'"}' | sed -n 's/.*"id":\([0-9]*\).*/\1/p')
[ -n "$kid" ] && pass "POST /knowledge-bases -> id=$kid" || fail "创建KB" "无id"
if [ -n "$kid" ]; then
  tmp=$(mktemp); printf 'term,cn\nfoo,福\nbar,巴\n' > "$tmp"
  v=$(curl -s -b "$COOKIEJAR" -F "file=@$tmp" "$API/knowledge-bases/$kid/versions")
  echo "$v" | grep -q '"version":1' && pass "POST versions 上传 -> v1" || fail "上传版本" "$v"
  # 上传第二版，删除旧版应 200；删除最新版应 400
  v2=$(curl -s -b "$COOKIEJAR" -F "file=@$tmp" "$API/knowledge-bases/$kid/versions")
  echo "$v2" | grep -q '"version":2' && pass "再传一版 -> v2" || fail "上传v2" "$v2"
  c=$(code -b "$COOKIEJAR" -X DELETE "$API/knowledge-bases/$kid/versions/2")
  [ "$c" = "400" ] && pass "删最新版 -> 400(保护)" || fail "删最新版" "实得$c(期望400)"
  c=$(code -b "$COOKIEJAR" -X DELETE "$API/knowledge-bases/$kid/versions/1")
  [ "$c" = "200" ] && pass "删旧版 -> 200" || fail "删旧版" "实得$c"
  rm -f "$tmp"
  c=$(code -b "$COOKIEJAR" -X DELETE "$API/knowledge-bases/$kid")
  [ "$c" = "200" ] && pass "删KB -> 200" || fail "删KB" "实得$c"
fi

echo "== 9. tasks 创建(multipart, url源)/列表/详情/阶段/结果 =="
tid=$(curl -s -b "$COOKIEJAR" -X POST "$API/tasks" \
  -F "video_url=$VIDEO_URL" \
  -F "summary_profile_id=0" -F "asr_profile_id=0" -F "vision_profile_id=0" \
  -F "reading_comprehension_profile_id=0" \
  -F "generate_ai_subtitle_via_asr=0" -F "generate_vision_subtitle=0" \
  -F "screenshot_mode=simple" -F "screenshot_interval=60" -F "reading_depth=2" \
  -F "summary_preference=" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
if [ -n "$tid" ]; then pass "POST /tasks -> id=$tid"; else fail "创建任务" "无id"; fi
if [ -n "$tid" ]; then
  [ "$(code -b "$COOKIEJAR" "$API/tasks")" = "200" ] && pass "GET /tasks 200" || fail "GET /tasks" ""
  [ "$(code -b "$COOKIEJAR" "$API/tasks/$tid")" = "200" ] && pass "GET /tasks/{id} 200" || fail "GET task" ""
  [ "$(code -b "$COOKIEJAR" "$API/tasks/$tid/stages")" = "200" ] && pass "GET stages 200" || fail "GET stages" ""
  r=$(json -b "$COOKIEJAR" "$API/tasks/$tid/result")
  for k in task cost subtitles knowledge_bases stages summary_md knowledge_md; do
    echo "$r" | grep -q "\"$k\"" && true || fail "result 缺键 $k" ""
  done
  pass "result 含 task/cost/subtitles/knowledge_bases/stages/summary_md/knowledge_md"
  [ "$(code -b "$COOKIEJAR" "$API/tasks/$tid/files/summary.md")" = "404" ] && pass "未生成 summary.md -> 404(符合)" || echo "  [INFO] summary.md 状态非404(可能已生成)"
  c=$(code -b "$COOKIEJAR" -X DELETE "$API/tasks/$tid")
  [ "$c" = "200" ] && pass "DELETE /tasks/{id} -> 200" || fail "删任务" "实得$c"
fi

echo "== 10. logout 清 Cookie =="
[ "$(code -b "$COOKIEJAR" -X POST "$API/logout")" = "200" ] && pass "POST /logout 200" || fail "logout" ""

rm -f "$COOKIEJAR"
echo
echo "================ 汇总: PASS=$PASS FAIL=$FAIL ================"
[ "$FAIL" -eq 0 ] && echo "全部通过" || echo "存在失败项，见上"
exit $FAIL
