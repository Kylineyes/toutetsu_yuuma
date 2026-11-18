<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { api } from '../api.js'
import PageHeader from '../components/PageHeader.vue'
import UiButton from '../components/UiButton.vue'

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
  if (form.value.temperature !== '' && form.value.temperature !== null) {
    const t = Number(form.value.temperature)
    if (Number.isNaN(t) || t < 0 || t > 2) return '温度须为 0~2 之间的数字，或留空使用默认'
  }
  const to = Number(form.value.timeout_seconds)
  if (!Number.isFinite(to) || to <= 0) return '超时须为正整数秒'
  const raw = (form.value.extra || '').trim()
  if (raw !== '') {
    let parsed
    try { parsed = JSON.parse(raw) } catch (e) { return 'extra 不是合法 JSON：' + e.message }
    if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) return 'extra 须为 JSON 对象'
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
async function del(id) {
  if (!confirm('删除该方案？')) return
  try { await api.deleteProfile(id); await load() } catch (e) { err.value = e.message }
}
onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader title="配置方案" eyebrow="能力配置" description="为总结、ASR、识图和阅读理解分别配置协议、模型与访问凭据。" />
    <div v-if="err" class="alert" role="alert"><p>{{ err }}</p></div>
    <section class="card" aria-labelledby="new-profile-title">
      <div class="card-header"><div><h2 id="new-profile-title" class="card-title">新增方案</h2><p class="card-description">每类能力独立配置；API Key 仅在创建时提交。</p></div></div>
      <form @submit.prevent="add">
        <div class="form-grid form-grid-three">
          <label class="field"><span class="field-label">能力</span><select v-model="form.capability"><option v-for="c in caps" :key="c" :value="c">{{ capLabel[c] }}</option></select></label>
          <label class="field"><span class="field-label">API 协议</span><select v-model="form.protocol"><option v-for="p in protoOptions" :key="p.v" :value="p.v">{{ p.t }}</option></select></label>
          <label class="field"><span class="field-label">名称</span><input v-model="form.name" placeholder="方案名称" /></label>
          <label class="field field-span"><span class="field-label">Base URL</span><input v-model="form.base_url" /></label>
          <label class="field"><span class="field-label">Model</span><input v-model="form.model" placeholder="模型标识" /></label>
          <label class="field"><span class="field-label">API Key</span><input v-model="form.api_key" type="password" autocomplete="off" /></label>
          <label class="field"><span class="field-label">Temperature</span><input v-model="form.temperature" type="number" min="0" max="2" step="0.1" placeholder="留空使用默认值" /></label>
          <label class="field"><span class="field-label">超时（秒）</span><input v-model="form.timeout_seconds" type="number" min="1" step="1" /></label>
          <label class="field field-span"><span class="field-label">extra（JSON）</span><textarea v-model="form.extra" class="cell-mono" rows="2" placeholder="{}" /></label>
        </div>
        <div class="form-actions"><UiButton type="submit" icon="plus">新增方案</UiButton></div>
      </form>
    </section>

    <section v-for="c in caps" :key="c" class="card card-flush profile-section" :aria-labelledby="'cap-' + c">
      <div class="card-header"><div><h2 :id="'cap-' + c" class="card-title">{{ capLabel[c] }}</h2><p class="card-description">{{ byCap(c).length }} 个方案</p></div><span class="tag tag-plain">{{ c }}</span></div>
      <div v-if="byCap(c).length" class="table-scroll">
        <table>
          <thead><tr><th>ID / 名称</th><th>协议</th><th>Model</th><th>Temp</th><th>超时</th><th>extra</th><th>Base URL</th><th><span class="sr-only">操作</span></th></tr></thead>
          <tbody>
            <tr v-for="p in byCap(c)" :key="p.id">
              <td><span class="cell-primary">#{{ p.id }} · {{ p.name }}</span></td>
              <td class="cell-secondary">{{ p.protocol }}</td>
              <td class="cell-mono">{{ p.model }}</td>
              <td class="numeric">{{ p.temperature ?? '—' }}</td>
              <td class="numeric">{{ p.timeout_seconds }}s</td>
              <td class="cell-mono extra-cell">{{ p.extra || '{}' }}</td>
              <td class="cell-secondary url-cell">{{ p.base_url }}</td>
              <td><UiButton variant="quiet-danger" size="small" @click="del(p.id)">删除</UiButton></td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="empty-state compact"><p>暂无方案。</p></div>
    </section>
  </div>
</template>

<style scoped>
.form-actions { display: flex; justify-content: flex-end; margin-top: 20px; }
.profile-section { margin-top: 16px; }
.url-cell { min-width: 180px; max-width: 280px; overflow-wrap: anywhere; }
.extra-cell { max-width: 180px; overflow-wrap: anywhere; }
.compact { padding: 24px; }
</style>
