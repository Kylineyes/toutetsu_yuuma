<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { api } from '../api.js'

const caps = ['summary', 'asr', 'vision', 'reading']
const capLabel = { summary: '总结', asr: 'ASR 语音识别', vision: '识图字幕生成', reading: '阅读理解' }
const chatProtocols = [
  { v: 'openai_chat', t: 'OpenAI Chat Completions' },
  { v: 'openai_responses', t: 'OpenAI Responses' },
  { v: 'anthropic_messages', t: 'Anthropic Messages' },
  { v: 'gemini', t: 'Gemini API' },
]
const asrProtocols = [
  { v: 'deepseek_asr', t: 'DeepSeek ASR (/v1/tools/asr)' },
  { v: 'openai_audio', t: 'OpenAI Audio (/v1/audio/transcriptions)' },
  { v: 'openai_whisper', t: 'OpenAI Whisper（兼容）' },
]
const profiles = ref([])
const err = ref('')
const form = ref(newForm('summary'))

function newForm(cap) {
  return { capability: cap, protocol: cap === 'asr' ? 'deepseek_asr' : 'openai_chat', name: '', base_url: 'https://api.deepseek.com', model: '', api_key: '', temperature: '', timeout_seconds: 180, extra: '{}' }
}
const protoOptions = computed(() => (form.value.capability === 'asr' ? asrProtocols : chatProtocols))
watch(() => form.value.capability, (c) => { form.value.protocol = c === 'asr' ? 'deepseek_asr' : 'openai_chat' })

async function load() { try { const list = await api.profiles(); profiles.value = Array.isArray(list) ? list : [] } catch (e) { err.value = e.message } }
function byCap(c) { return profiles.value.filter((p) => p.capability === c) }
function validate() {
  // 温度：空=默认；否则须为 0~2 的数字
  if (form.value.temperature !== '' && form.value.temperature !== null) {
    const t = Number(form.value.temperature)
    if (Number.isNaN(t) || t < 0 || t > 2) return '温度须为 0~2 之间的数字，或留空使用默认'
  }
  // 超时：须为正整数秒
  const to = Number(form.value.timeout_seconds)
  if (!Number.isFinite(to) || to <= 0) return '超时须为正整数秒'
  // extra：须为合法 JSON 对象
  const raw = (form.value.extra || '').trim()
  if (raw !== '') {
    let parsed
    try { parsed = JSON.parse(raw) } catch (e) { return 'extra 不是合法 JSON：' + e.message }
    if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) return 'extra 须为 JSON 对象，如 {"k":1}'
  }
  return ''
}
async function add() {
  err.value = ''
  const v = validate()
  if (v) { err.value = v; return }
  const payload = { ...form.value }
  payload.temperature = form.value.temperature === '' || form.value.temperature === null ? null : Number(form.value.temperature)
  payload.timeout_seconds = Number(form.value.timeout_seconds)
  payload.extra = (form.value.extra || '').trim() || '{}'
  try {
    await api.createProfile(payload)
    form.value = newForm(form.value.capability)
    await load()
  } catch (e) { err.value = e.message }
}
async function del(id) { if (!confirm('删除该方案？')) return; try { await api.deleteProfile(id); await load() } catch (e) { err.value = e.message } }
onMounted(load)
</script>

<template>
  <h2>配置方案（Profile）</h2>
  <p style="color:#64748b">四类能力各自独立配置，可各用不同 endpoint / api_key / model / 协议。建任务时每类各选一个。</p>
  <div v-if="err" class="card err" style="color:#b91c1c">{{ err }}</div>

  <div class="card">
    <div class="row">
      <label class="fld">能力
        <select v-model="form.capability">
          <option v-for="c in caps" :key="c" :value="c">{{ capLabel[c] }}</option>
        </select>
      </label>
      <label class="fld">API 协议
        <select v-model="form.protocol">
          <option v-for="p in protoOptions" :key="p.v" :value="p.v">{{ p.t }}</option>
        </select>
      </label>
      <label class="fld">名称<input v-model="form.name" placeholder="如 deepseek-chat" /></label>
      <label class="fld">Base URL<input v-model="form.base_url" /></label>
      <label class="fld">Model<input v-model="form.model" placeholder="如 deepseek-flash" /></label>
      <label class="fld">API Key<input v-model="form.api_key" type="password" /></label>
      <label class="fld">Temperature<input v-model="form.temperature" type="number" step="0.1" placeholder="空=默认" style="width:110px" /></label>
      <label class="fld">超时(秒)<input v-model="form.timeout_seconds" type="number" style="width:100px" /></label>
      <label class="fld">extra(JSON)<input v-model="form.extra" placeholder="{}" style="width:160px" /></label>
    </div>
    <button @click="add">新增方案</button>
  </div>

  <div v-for="c in caps" :key="c" class="card">
    <h3>{{ capLabel[c] }}</h3>
    <table v-if="byCap(c).length">
      <thead><tr><th>ID</th><th>名称</th><th>协议</th><th>Model</th><th>Temp</th><th>超时</th><th>extra</th><th>Base URL</th><th></th></tr></thead>
      <tbody>
        <tr v-for="p in byCap(c)" :key="p.id">
          <td>{{ p.id }}</td><td>{{ p.name }}</td><td>{{ p.protocol }}</td><td>{{ p.model }}</td>
          <td>{{ p.temperature ?? '-' }}</td><td>{{ p.timeout_seconds }}s</td><td style="font-family:monospace;font-size:12px">{{ p.extra || '{}' }}</td><td>{{ p.base_url }}</td>
          <td><button class="danger" @click="del(p.id)">删除</button></td>
        </tr>
      </tbody>
    </table>
    <p v-else style="color:#94a3b8">暂无</p>
  </div>
</template>
