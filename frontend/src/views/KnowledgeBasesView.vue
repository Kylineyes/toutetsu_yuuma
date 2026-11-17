<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const kbs = ref([])
const err = ref('')
const newName = ref('')
const versions = ref({}) // kbId -> list
const uploadFiles = ref({}) // kbId -> File

async function load() {
  try { kbs.value = await api.kbs() } catch (e) { err.value = e.message }
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
  try { versions.value[kb.id] = await api.kbVersions(kb.id) } catch (e) { err.value = e.message }
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
    versions.value[kbId] = await api.kbVersions(kbId)
  } catch (e) { err.value = e.message }
}
onMounted(load)
</script>

<template>
  <h2>知识库</h2>
  <p style="color:#64748b">术语正文存本地目录，DB 只做版本索引。每次上传/更新刷新一个版本（v1、v2…，快照不可变）。</p>
  <div v-if="err" class="card" style="color:#b91c1c">{{ err }}</div>

  <div class="card">
    <div class="row">
      <label class="fld">新知识库名称<input v-model="newName" placeholder="如 葡萄酒术语库" /></label>
      <button @click="create">创建</button>
    </div>
  </div>

  <div v-for="kb in kbs" :key="kb.id" class="card">
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
      <thead><tr><th>版本</th><th>文件</th><th>术语数</th><th>sha256</th><th>时间</th></tr></thead>
      <tbody>
        <tr v-for="v in versions[kb.id]" :key="v.id">
          <td>v{{ v.version }}</td><td>{{ v.file_name }}</td><td>{{ v.term_count }}</td>
          <td style="font-family:monospace;font-size:11px">{{ (v.sha256||'').slice(0,16) }}…</td><td>{{ v.created_at }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
