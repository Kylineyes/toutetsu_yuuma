# V1 契约复检脚本（PowerShell 版，Windows 原生）—— 待后端可运行且 lead 通知后执行
# 与 bash 版 v1_contract_check.sh 等价，覆盖 plan.md §6 全部鉴权与 CRUD 流程。
# 用法: $env:BASE='http://127.0.0.1:8080'; $env:PASSWORD='changeme'; $env:API_TOKEN='secret'; ./v1_contract_check.ps1
$ErrorActionPreference = 'Continue'
$BASE = 'http://127.0.0.1:8080'
if ($env:BASE) { $BASE = $env:BASE }
$API = "$BASE/api/v1"
$PASSWORD = 'changeme'
if ($env:PASSWORD) { $PASSWORD = $env:PASSWORD }
$API_TOKEN = $env:API_TOKEN
$VIDEO_URL = 'https://example.com/test.mp4'
if ($env:VIDEO_URL) { $VIDEO_URL = $env:VIDEO_URL }
$script:PASS = 0
$script:FAIL = 0
function P([string]$m) { Write-Host "  [PASS] $m"; $script:PASS++ }
function F([string]$m, [string]$d) { Write-Host "  [FAIL] $m -> $d"; $script:FAIL++ }
function Req([string]$url, [string]$method, $body, $contentType, $headers, $sess) {
  $p = @{ Uri = $url; Method = $method; UseBasicParsing = $true; SkipHttpErrorCheck = $true }
  if ($body) { $p.Body = $body }
  if ($contentType) { $p.ContentType = $contentType }
  if ($headers) { $p.Headers = $headers }
  if ($sess) { $p.WebSession = $sess }
  try {
    $resp = Invoke-WebRequest @p
    return @{ Code = [int]$resp.StatusCode; Body = $resp.Content }
  } catch {
    return @{ Code = [int]$_.Exception.Response.StatusCode; Body = '' }
  }
}
Write-Host "== 目标: $BASE =="
Write-Host '== 1. 免鉴权端点 =='
if ((Req "$API/health" 'GET' $null $null $null $null).Code -eq 200) { P 'GET /health 200' } else { F 'GET /health' '非200' }
if ((Req "$API/session" 'GET' $null $null $null $null).Code -eq 200) { P 'GET /session 200' } else { F 'GET /session' '非200' }
Write-Host '== 2. 401 未授权 =='
$c = (Req "$API/profiles" 'GET' $null $null $null $null).Code
if ($c -eq 401) { P 'GET /profiles 无凭证 -> 401' } else { F 'GET /profiles 无凭证' ('实得 ' + $c) }
Write-Host '== 3. 错误口令 =='
$bad = '{"password":"__wrong__"}'
if ((Req "$API/login" 'POST' $bad 'application/json' $null $null).Code -eq 401) { P 'login 错误口令 -> 401' } else { F 'login 错误口令' '' }
Write-Host '== 4. 正确口令 + Cookie =='
$good = '{"password":"' + $PASSWORD + '"}'
$sess = $null
$loginResp = $null
try { $loginResp = Invoke-WebRequest -Uri "$API/login" -Method POST -Body $good -ContentType 'application/json' -UseBasicParsing -SessionVariable sess -SkipHttpErrorCheck } catch { }
if ($loginResp -and [int]$loginResp.StatusCode -eq 200) { P 'login 正确口令 -> 200' } else { F 'login 正确口令' '' }
$hasCookie = $false
if ($sess) { $jar = $sess.Cookies.GetCookies($BASE); foreach ($ck in $jar) { if ($ck.Name -eq 'session') { $hasCookie = $true } } }
if ($hasCookie) { P '含 session Cookie' } else { F 'Cookie' '无 session' }
Write-Host '== 5. Cookie 鉴权 =='
if ((Req "$API/profiles" 'GET' $null $null $null $sess).Code -eq 200) { P 'GET /profiles 带Cookie -> 200' } else { F 'GET /profiles Cookie' '' }
Write-Host '== 6. X-API-Token =='
if ($API_TOKEN) {
  $h = @{ 'X-API-Token' = $API_TOKEN }
  if ((Req "$API/profiles" 'GET' $null $null $h $null).Code -eq 200) { P 'X-API-Token 正确 -> 200' } else { F 'X-API-Token' '' }
  $hw = @{ 'X-API-Token' = 'wrong' }
  if ((Req "$API/profiles" 'GET' $null $null $hw $null).Code -eq 401) { P 'X-API-Token 错误 -> 401' } else { F '错误Token' '' }
} else { Write-Host '  [SKIP] 未提供 API_TOKEN' }
Write-Host '== 7. profiles CRUD + 脱敏 =='
$pjson = '{"capability":"summary","protocol":"openai_chat","name":"v1-test","base_url":"https://api.deepseek.com","model":"deepseek-chat","api_key":"sk-x","temperature":0.3,"timeout_seconds":120,"extra":"{}"}'
$pr = Req "$API/profiles" 'POST' $pjson 'application/json' $null $sess
$pid = 0
if ($pr.Body -match '"id":(d+)') { $pid = [int]$Matches[1] }
if ($pid -gt 0) { P ('POST /profiles -> id=' + $pid) } else { F 'POST /profiles' $pr.Body }
if ($pid -gt 0) {
  $g = Req "$API/profiles/$pid" 'GET' $null $null $null $sess
  if ($g.Body -match '"api_key":"***"') { P 'api_key 脱敏' } else { F 'api_key 脱敏' $g.Body }
  if ((Req "$API/profiles/$pid" 'DELETE' $null $null $null $sess).Code -eq 200) { P 'DELETE profile 200' } else { F 'DELETE profile' '' }
}
Write-Host '== 8. knowledge-bases 版本与删除保护 =='
$kbjson = '{"name":"v1-kb-' + (Get-Random) + '"}'
$kr = Req "$API/knowledge-bases" 'POST' $kbjson 'application/json' $null $sess
$kid = 0
if ($kr.Body -match '"id":(d+)') { $kid = [int]$Matches[1] }
if ($kid -gt 0) { P ('POST /knowledge-bases -> id=' + $kid) } else { F '创建KB' $kr.Body }
if ($kid -gt 0) {
  $tmp = [System.IO.Path]::GetTempFileName()
  $content = "term,cn" + [Environment]::NewLine + "foo,fu" + [Environment]::NewLine + "bar,ba"
  Set-Content -Path $tmp -Value $content -Encoding UTF8
  $v1r = Invoke-WebRequest -Uri "$API/knowledge-bases/$kid/versions" -Method POST -Form @{ file = (Get-Item $tmp) } -UseBasicParsing -WebSession $sess -SkipHttpErrorCheck
  if ($v1r.Content -match '"version":1') { P '上传 v1' } else { F '上传v1' $v1r.Content }
  $v2r = Invoke-WebRequest -Uri "$API/knowledge-bases/$kid/versions" -Method POST -Form @{ file = (Get-Item $tmp) } -UseBasicParsing -WebSession $sess -SkipHttpErrorCheck
  if ($v2r.Content -match '"version":2') { P '上传 v2' } else { F '上传v2' $v2r.Content }
  if ((Req "$API/knowledge-bases/$kid/versions/2" 'DELETE' $null $null $null $sess).Code -eq 400) { P '删最新版 -> 400' } else { F '删最新版' '' }
  if ((Req "$API/knowledge-bases/$kid/versions/1" 'DELETE' $null $null $null $sess).Code -eq 200) { P '删旧版 -> 200' } else { F '删旧版' '' }
  Remove-Item $tmp -ErrorAction SilentlyContinue
  if ((Req "$API/knowledge-bases/$kid" 'DELETE' $null $null $null $sess).Code -eq 200) { P '删KB 200' } else { F '删KB' '' }
}
Write-Host '== 9. tasks 创建/列表/详情/阶段/结果 =='
$form = @{ video_url = $VIDEO_URL; summary_profile_id = '0'; asr_profile_id = '0'; vision_profile_id = '0'; reading_comprehension_profile_id = '0'; generate_ai_subtitle_via_asr = '0'; generate_vision_subtitle = '0'; screenshot_mode = 'simple'; screenshot_interval = '60'; reading_depth = '2'; summary_preference = '' }
$createResp = $null
try { $createResp = Invoke-WebRequest -Uri "$API/tasks" -Method POST -Form $form -UseBasicParsing -WebSession $sess -SkipHttpErrorCheck } catch { }
$tid = ''
if ($createResp -and $createResp.Content -match '"id":"([^"]+)"') { $tid = $Matches[1] }
if ($tid) { P ('POST /tasks -> id=' + $tid) } else { $d = ''; if ($createResp) { $d = $createResp.Content }; F '创建任务' $d }
if ($tid) {
  if ((Req "$API/tasks" 'GET' $null $null $null $sess).Code -eq 200) { P 'GET /tasks 200' } else { F 'GET /tasks' '' }
  if ((Req "$API/tasks/$tid" 'GET' $null $null $null $sess).Code -eq 200) { P 'GET task 200' } else { F 'GET task' '' }
  if ((Req "$API/tasks/$tid/stages" 'GET' $null $null $null $sess).Code -eq 200) { P 'GET stages 200' } else { F 'GET stages' '' }
  $res = (Req "$API/tasks/$tid/result" 'GET' $null $null $null $sess).Body
  $missing = @()
  foreach ($k in @('task','cost','subtitles','knowledge_bases','stages','summary_md','knowledge_md')) {
    $needle = '"' + $k + '"'
    if ($res.IndexOf($needle) -lt 0) { $missing += $k }
  }
  if ($missing.Count -eq 0) { P 'result 七键齐' } else { F 'result 缺键' ($missing -join ',') }
  if ((Req "$API/tasks/$tid/files/summary.md" 'GET' $null $null $null $sess).Code -eq 404) { P '未生成 summary.md -> 404' } else { Write-Host '  [INFO] summary.md 非404(可能已生成)' }
  if ((Req "$API/tasks/$tid" 'DELETE' $null $null $null $sess).Code -eq 200) { P 'DELETE task 200' } else { F '删任务' '' }
}
Write-Host '== 10. logout =='
if ((Req "$API/logout" 'POST' $null $null $null $sess).Code -eq 200) { P 'logout 200' } else { F 'logout' '' }
Write-Host ''
Write-Host ("================ 汇总: PASS=" + $PASS + " FAIL=" + $FAIL + " ================")
exit $FAIL