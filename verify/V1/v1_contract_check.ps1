# V1 契约复检（PowerShell + 原生 curl.exe；已在 Windows PowerShell 5.1 实测通过，26/26 PASS）
# 用法：先启动后端(设 PASSWORD+API_TOKEN)，再：
#   $env:BASE='http://127.0.0.1:8080'; $env:PASSWORD='...'; $env:API_TOKEN='...'; powershell -ExecutionPolicy Bypass -File verify/V1/v1_contract_check.ps1
# 说明：为兼容 PowerShell 5.1，全部 HTTP 经由 curl.exe；JSON 体经临时文件 --data-binary 传递（规避 PS 参数转义问题）。
﻿# V1 契约复检（PowerShell + 原生 curl.exe，兼容 PS5.1）
$ErrorActionPreference = 'Continue'
$BASE = 'http://127.0.0.1:18081'
if ($env:BASE) { $BASE = $env:BASE }
$API = "$BASE/api/v1"
$PASSWORD = 'test123'; if ($env:PASSWORD) { $PASSWORD = $env:PASSWORD }
$API_TOKEN = $env:API_TOKEN
$VIDEO_URL = 'https://example.com/test.mp4'
$TMP = Join-Path $env:TEMP ('v1_' + [guid]::NewGuid().ToString('N')); New-Item -ItemType Directory -Force -Path $TMP | Out-Null
$COOKIE = Join-Path $TMP 'cookies.txt'
$script:BODYN=0
function JsonBody([string]$json){
  $script:BODYN++
  $f = Join-Path $TMP ('req' + $script:BODYN + '.json')
  [System.IO.File]::WriteAllText($f, $json)
  return ('@' + $f)
}
$script:PASS=0; $script:FAIL=0
function P([string]$m){ Write-Output "  [PASS] $m"; $script:PASS++ }
function F([string]$m,[string]$d){ Write-Output "  [FAIL] $m -> $d"; $script:FAIL++ }
function DoCurl([string]$method,[string]$url,$extra){
  $a = @('-s','-o',(Join-Path $TMP 'body.txt'),'-w','%{http_code}','-X',$method,'-b',$COOKIE,'-c',$COOKIE)
  if ($extra) { foreach($e in $extra){ $a += $e } }
  $a += $url
  $code = & curl.exe @a 2>$null
  $bf = Join-Path $TMP 'body.txt'
  $global:LASTBODY = ''
  if (Test-Path $bf) { $global:LASTBODY = [System.IO.File]::ReadAllText($bf) }
  return [int]$code
}
function DoCurlNC([string]$method,[string]$url,$extra){
  $a = @('-s','-o',(Join-Path $TMP 'body_nc.txt'),'-w','%{http_code}','-X',$method)
  if ($extra) { foreach($e in $extra){ $a += $e } }
  $a += $url
  $code = & curl.exe @a 2>$null
  $bf = Join-Path $TMP 'body_nc.txt'
  $global:LASTBODY = ''
  if (Test-Path $bf) { $global:LASTBODY = [System.IO.File]::ReadAllText($bf) }
  return [int]$code
}
Write-Output "== 目标: $BASE =="
Write-Output '== 1. 免鉴权端点 =='
if ((DoCurl 'GET' "$API/health" $null) -eq 200){ P 'GET /health 200' } else { F 'GET /health' '非200' }
if ((DoCurl 'GET' "$API/session" $null) -eq 200){ P 'GET /session 200' } else { F 'GET /session' '非200' }
Write-Output '== 2. 401 未授权 =='
$c = DoCurl 'GET' "$API/profiles" $null
if ($c -eq 401){ P 'GET /profiles 无凭证 -> 401' } else { F 'GET /profiles 无凭证' "实得 $c" }
Write-Output '== 3. 错误口令 =='
$c = DoCurl 'POST' "$API/login" @('-H','Content-Type: application/json','--data-binary',(JsonBody '{"password":"__wrong__"}'))
if ($c -eq 401){ P 'POST /login 错误口令 -> 401' } else { F 'login 错误口令' "实得 $c" }
Write-Output '== 4. 正确口令 + Cookie =='
$good = '{"password":"' + $PASSWORD + '"}'
$c = DoCurl 'POST' "$API/login" @('-H','Content-Type: application/json','--data-binary',(JsonBody $good))
if ($c -eq 200){ P 'POST /login 正确口令 -> 200' } else { F 'login 正确口令' "实得 $c" }
$cookietxt = ''; if (Test-Path $COOKIE) { $cookietxt = Get-Content $COOKIE -Raw }
if ($cookietxt -match 'session'){ P 'Set-Cookie 含 session' } else { F 'Cookie' '无 session' }
Write-Output '== 5. Cookie 鉴权 =='
if ((DoCurl 'GET' "$API/profiles" $null) -eq 200){ P 'GET /profiles 带Cookie -> 200' } else { F 'GET profiles Cookie' '' }
Write-Output '== 6. X-API-Token =='
if ($API_TOKEN){
  if ((DoCurl 'GET' "$API/profiles" @('-H',("X-API-Token: " + $API_TOKEN))) -eq 200){ P 'X-API-Token 正确 -> 200' } else { F 'X-API-Token' '' }
  $wtok = DoCurlNC 'GET' "$API/profiles" @('-H','X-API-Token: wrong')
  if ($wtok -eq 401){ P 'X-API-Token 错误 -> 401(无Cookie)' } else { F '错误Token' "实得 $wtok" }
} else { Write-Output '  [SKIP] 无 API_TOKEN' }
Write-Output '== 7. profiles CRUD + 脱敏 =='
$pjson = '{"capability":"summary","protocol":"openai_chat","name":"v1-test","base_url":"https://api.deepseek.com","model":"deepseek-chat","api_key":"sk-x","temperature":0.3,"timeout_seconds":120,"extra":"{}"}'
$c = DoCurl 'POST' "$API/profiles" @('-H','Content-Type: application/json','--data-binary',(JsonBody $pjson))
$profId = 0
if ($global:LASTBODY -match '"id":([0-9]+)'){ $profId = [int]$Matches[1] }
if ($c -eq 201 -and $profId -gt 0){ P "POST /profiles -> id=$profId" } else { F 'POST profiles' "code=$c body=$global:LASTBODY" }
if ($profId -gt 0){
  $c = DoCurl 'GET' "$API/profiles/$profId" $null
  if (($global:LASTBODY.IndexOf('api_key') -ge 0) -and ($global:LASTBODY.IndexOf('sk-x') -lt 0)){ P 'api_key 已脱敏(不含明文sk-x)' } else { F 'api_key 脱敏' $global:LASTBODY }
  if ((DoCurl 'DELETE' "$API/profiles/$profId" $null) -eq 200){ P 'DELETE profile 200' } else { F 'DELETE profile' '' }
}
Write-Output '== 8. knowledge-bases 版本与删除保护 =='
$kbjson = '{"name":"v1-kb-' + (Get-Random) + '"}'
$c = DoCurl 'POST' "$API/knowledge-bases" @('-H','Content-Type: application/json','--data-binary',(JsonBody $kbjson))
$kid = 0
if ($global:LASTBODY -match '"id":([0-9]+)'){ $kid = [int]$Matches[1] }
if ($c -eq 201 -and $kid -gt 0){ P "POST /knowledge-bases -> id=$kid" } else { F '创建KB' "code=$c body=$global:LASTBODY" }
if ($kid -gt 0){
  $termFile = Join-Path $TMP 'terms.csv'
  $nl = [Environment]::NewLine
  [System.IO.File]::WriteAllText($termFile, ('term,cn' + $nl + 'foo,fu' + $nl + 'bar,ba' + $nl))
  $c = DoCurl 'POST' "$API/knowledge-bases/$kid/versions" @('-F',("file=@" + $termFile))
  if ($c -eq 201 -and $global:LASTBODY -match '"version":1'){ P '上传 v1 (201)' } else { F '上传v1' "code=$c body=$global:LASTBODY" }
  $c = DoCurl 'POST' "$API/knowledge-bases/$kid/versions" @('-F',("file=@" + $termFile))
  if ($c -eq 201 -and $global:LASTBODY -match '"version":2'){ P '上传 v2 (201)' } else { F '上传v2' "code=$c body=$global:LASTBODY" }
  $c = DoCurl 'DELETE' "$API/knowledge-bases/$kid/versions/2" $null
  if ($c -eq 400){ P '删最新版 -> 400(保护)' } else { F '删最新版' "实得 $c(期望400)" }
  $c = DoCurl 'DELETE' "$API/knowledge-bases/$kid/versions/1" $null
  if ($c -eq 200){ P '删旧版 -> 200' } else { F '删旧版' "实得 $c" }
  $c = DoCurl 'DELETE' "$API/knowledge-bases/$kid" $null
  if ($c -eq 200){ P '删KB -> 200' } else { F '删KB' "实得 $c" }
}
Write-Output '== 9. tasks 创建/列表/详情/stages/result =='
$c = DoCurl 'POST' "$API/tasks" @('-F',("video_url=" + $VIDEO_URL),'-F','summary_profile_id=0','-F','asr_profile_id=0','-F','vision_profile_id=0','-F','reading_comprehension_profile_id=0','-F','generate_ai_subtitle_via_asr=0','-F','generate_vision_subtitle=0','-F','screenshot_mode=simple','-F','screenshot_interval=60','-F','reading_depth=2','-F','summary_preference=')
$tid = ''
if ($global:LASTBODY -match '"id":"([^"]+)"'){ $tid = $Matches[1] }
if ($c -eq 201 -and $tid){ P "POST /tasks -> id=$tid" } else { F '创建任务' "code=$c body=$global:LASTBODY" }
if ($tid){
  if ((DoCurl 'GET' "$API/tasks" $null) -eq 200){ P 'GET /tasks 200' } else { F 'GET tasks' '' }
  if ((DoCurl 'GET' "$API/tasks/$tid" $null) -eq 200){ P 'GET task 200' } else { F 'GET task' '' }
  if ((DoCurl 'GET' "$API/tasks/$tid/stages" $null) -eq 200){ P 'GET stages 200' } else { F 'GET stages' '' }
  $c = DoCurl 'GET' "$API/tasks/$tid/result" $null
  $missing = @()
  foreach ($k in @('task','cost','subtitles','knowledge_bases','stages','summary_md','knowledge_md')){ if ($global:LASTBODY.IndexOf('"' + $k + '"') -lt 0){ $missing += $k } }
  if ($missing.Count -eq 0){ P 'result 七键齐' } else { F 'result 缺键' ($missing -join ',') }
  $c = DoCurl 'GET' "$API/tasks/$tid/files/summary.md" $null
  if ($c -eq 404){ P '未生成 summary.md -> 404' } else { Write-Output "  [INFO] summary.md 状态=$c(可能已生成)" }
  if ((DoCurl 'DELETE' "$API/tasks/$tid" $null) -eq 200){ P 'DELETE task 200' } else { F '删任务' '' }
}
Write-Output '== 10. logout =='
if ((DoCurl 'POST' "$API/logout" $null) -eq 200){ P 'POST /logout 200' } else { F 'logout' '' }
Remove-Item -Recurse -Force $TMP -ErrorAction SilentlyContinue
Write-Output ''
Write-Output "================ 汇总: PASS=$PASS FAIL=$FAIL ================"
exit $FAIL