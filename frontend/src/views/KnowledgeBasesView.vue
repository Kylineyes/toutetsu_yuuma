<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api.js'

const kbs = ref([])
const err = ref('')
const newName = ref('')
const versions = ref({}) // kbId -> list
const uploadFiles = ref({}) // kbId -> File
const query = ref('') // 名称搜索关键字
const filteredKbs = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return kbs.value
  return kbs.value.filter(kb => String(kb.name || '').toLowerCase().includes(q) || (('#' + kb.id).includes(q)) || String(kb.id) === q)
})

async function load() {
  try { const list = await api.kbs(); kbs.value = Array.isArray(list) ? list : [] } catch (e) { err.value = e.message }
}
async function create() {
  if (!newName.value.trim()) return
  try { await api.createKB(newName.value.trim()); newName.value = ''; await load() } catch (e) { err.value = e.message }
}
async function del(id) {
  if (!confirm('删除该知识库？')) return
  try { await api.deleteKB(id); await load() } catch (e) { err.value = e.message }
}
async function toggleVersions(kb) {
  if (versions.value[kb.id]) { delete versions.value[kb.id]; return }
  try { const vs = await api.kbVersions(kb.id); versions.value[kb.id] = Array.isArray(vs) ? vs : [] } catch (e) { err.value = e.message }
}
function pickFile(kbId, e) { uploadFiles.value[kbId] = e.target.files[0] }
async function upload(kbId) {
  const f = uploadFiles.value[kbId]
  if (!f) { err.value = '请选择术语文件（CSV/JSON/TXT）'; return }
  const fd = new FormData(); fd.append('file', f)
  try {
    await api.uploadKB(kbId, fd)
    uploadFiles.value[kbId] = null
    await load()
    const vs2 = await api.kbVersions(kbId); versions.value[kbId] = Array.isArray(vs2) ? vs2 : []
  } catch (e) { err.value = e.message }
}
async function delVersion(kb, v) {
  if (!confirm('删除版本 v' + v.version + '？仅未被任务引用的旧版本可删。')) return
  err.value = ''
  try { await api.deleteKBVersion(kb.id, v.version); const vs3 = await api.kbVersions(kb.id); versions.value[kb.id] = Array.isArray(vs3) ? vs3 : []; await load() } catch (e) { err.value = e.message }
}
// 人类可读的文件大小
function fmtSize(n) {
  if (n == null) return '-'
  if (n < 1024) return n + ' B'
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB'
  return (n / 1024 / 1024).toFixed(2) + ' MB'
}
function dlVersion(kb, v) { return api.kbVersionFileUrl(kb.id, v.version) }
onMounted(load)
</script>

<template>
  <h2>知识库</h2>
  <p style="color:#64748b">术语正文存本地目录，DB 只做版本索引。每次上传/更新刷新一个版本（v1、v2…，快照不可变）。</p>
  <div v-if="err" class="card err" style="color:#b91c1c">{{ err }}</div>

  <div class="card">
    <div class="row">
      <label class="fld">新知识库名称<input v-model="newName" placeholder="如 葡萄酒术语库" /></label>
      <button @click="create">创建</button>
    </div>
  </div>

  <div class="card">
    <div class="row">
      <label class="fld" style="flex:1">搜索知识库名称<input v-model="query" placeholder="输入名称关键字筛选" /></label>
      <span style="color:#64748b;align-self:flex-end">共 {{ kbs.length }} 个，匹配 {{ filteredKbs.length }} 个</span>
    </div>
  </div>

  <div v-if="query.trim() && filteredKbs.length === 0" class="card" style="color:#64748b">没有匹配「{{ query }}」的知识库</div>

  <div v-for="kb in filteredKbs" :key="kb.id" class="card">
    <div class="row" style="justify-content:space-between">
      <div><strong>#{{ kb.id }} {{ kb.name }}</strong> <span class="tag">最新 v{{ kb.latest_version }}</span></div>
      <div class="row" style="margin:0">
        <input type="file" @change="e => pickFile(kb.id, e)" />
        <button @click="upload(kb.id)">上传新版本</button>
        <button class="ghost" @click="toggleVersions(kb)">版本</button>
        <button class="danger" @click="del(kb.id)">删除</button>
      </div>
    </div>
    <table v-if="versions[kb.id]">
      <thead><tr><th>版本</th><th>文件</th><th>术语数</th><th>大小</th><th>sha256</th><th>时间</th><th>操作</th></tr></thead>
      <tbody>
        <tr v-for="v in versions[kb.id]" :key="v.id">
          <td>v{{ v.version }} <span v-if="v.version === kb.latest_version" class="tag done">最新</span></td><td>{{ v.file_name }}</td><td>{{ v.term_count }}</td>
          <td>{{ fmtSize(v.size_bytes) }}</td>
          <td style="font-family:monospace;font-size:11px">{{ (v.sha256||'').slice(0,16) }}…</td><td>{{ v.created_at }}</td>
          <td class="row" style="margin:0;gap:6px"><a :href="dlVersion(kb, v)" target="_blank"><button class="ghost">下载</button></a><button class="danger" :disabled="v.version === kb.latest_version" @click="delVersion(kb, v)">删除</button></td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
