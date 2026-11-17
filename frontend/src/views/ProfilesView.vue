<script setup>
import { ref, onMounted, computed } from 'vue'
import { api } from '../api.js'

const caps = ['summary', 'asr', 'vision', 'reading']
const capLabel = { summary: '总结', asr: 'ASR 语音识别', vision: '识图字幕生成', reading: '阅读理解' }
const profiles = ref([])
const err = ref('')
const form = ref({ capability: 'summary', name: '', base_url: 'https://api.deepseek.com', model: '', api_key: '', extra: '' })

async function load() {
  try { profiles.value = await api.profiles() } catch (e) { err.value = e.message }
}
function byCap(c) { return profiles.value.filter(p => p.capability === c) }
async function add() {
  err.value = ''
  try {
    await api.createProfile(form.value)
    form.value = { capability: form.value.capability, name: '', base_url: 'https://api.deepseek.com', model: '', api_key: '', extra: '' }
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
  <h2>配置方案（Profile）</h2>
  <p style="color:#64748b">四类能力各自独立配置，可各用不同 endpoint / api_key / model。建任务时每类各选一个。</p>
  <div v-if="err" class="card" style="color:#b91c1c">{{ err }}</div>

  <div class="card">
    <div class="row">
      <label class="fld">能力
        <select v-model="form.capability">
          <option v-for="c in caps" :key="c" :value="c">{{ capLabel[c] }}</option>
        </select>
      </label>
      <label class="fld">名称<input v-model="form.name" placeholder="如 deepseek-asr" /></label>
      <label class="fld">Base URL<input v-model="form.base_url" /></label>
      <label class="fld">Model<input v-model="form.model" placeholder="如 deepseek-flash" /></label>
      <label class="fld">API Key<input v-model="form.api_key" type="password" /></label>
    </div>
    <button @click="add">新增方案</button>
  </div>

  <div v-for="c in caps" :key="c" class="card">
    <h3>{{ capLabel[c] }}</h3>
    <table v-if="byCap(c).length">
      <thead><tr><th>ID</th><th>名称</th><th>Model</th><th>Base URL</th><th>Key</th><th></th></tr></thead>
      <tbody>
        <tr v-for="p in byCap(c)" :key="p.id">
          <td>{{ p.id }}</td><td>{{ p.name }}</td><td>{{ p.model }}</td><td>{{ p.base_url }}</td><td>{{ p.api_key }}</td>
          <td><button class="danger" @click="del(p.id)">删除</button></td>
        </tr>
      </tbody>
    </table>
    <p v-else style="color:#94a3b8">暂无</p>
  </div>
</template>
