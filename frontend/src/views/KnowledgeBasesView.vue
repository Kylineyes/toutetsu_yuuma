<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api.js'
import PageHeader from '../components/PageHeader.vue'
import StatusTag from '../components/StatusTag.vue'
import UiButton from '../components/UiButton.vue'
import UiIcon from '../components/UiIcon.vue'

const kbs = ref([])
const err = ref('')
const newName = ref('')
const versions = ref({})
const uploadFiles = ref({})
const query = ref('')
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
function fmtSize(n) {
  if (n == null) return '—'
  if (n < 1024) return n + ' B'
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB'
  return (n / 1024 / 1024).toFixed(2) + ' MB'
}
function dlVersion(kb, v) { return api.kbVersionFileUrl(kb.id, v.version) }
onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader title="知识库" eyebrow="长期资产" description="维护可版本化的领域术语，在任务中用于纠错和热词注入。" />
    <div v-if="err" class="alert" role="alert"><p>{{ err }}</p></div>
    <section class="card" aria-labelledby="create-kb-title">
      <div class="card-header"><div><h2 id="create-kb-title" class="card-title">创建知识库</h2><p class="card-description">正文保存在本地目录，每次上传形成不可变版本快照。</p></div></div>
      <form class="toolbar" @submit.prevent="create">
        <label class="field"><span class="field-label">知识库名称</span><input v-model="newName" placeholder="输入领域或用途名称" /></label>
        <UiButton type="submit" icon="plus">创建</UiButton>
      </form>
    </section>

    <section class="card library-card" aria-labelledby="library-title">
      <div class="card-header library-header">
        <div><h2 id="library-title" class="card-title">知识库目录</h2><p class="card-description">共 {{ kbs.length }} 个，当前匹配 {{ filteredKbs.length }} 个</p></div>
        <label class="field search-field"><span class="field-label">搜索名称或 ID</span><input v-model="query" type="search" placeholder="输入关键字筛选" /></label>
      </div>
      <div v-if="query.trim() && filteredKbs.length === 0" class="empty-state"><UiIcon name="search" /><p>没有匹配「{{ query }}」的知识库。</p></div>
      <div v-else-if="!kbs.length" class="empty-state"><UiIcon name="knowledge" /><p>暂无知识库，请先创建。</p></div>
      <div v-else class="kb-directory">
        <article v-for="kb in filteredKbs" :key="kb.id" class="kb-item">
          <div class="kb-item-header">
            <div class="kb-identity">
              <span class="kb-icon"><UiIcon name="knowledge" /></span>
              <div><h3>{{ kb.name }}</h3><p><span class="cell-mono">#{{ kb.id }}</span> · 最新 v{{ kb.latest_version }}</p></div>
            </div>
            <UiButton variant="quiet-danger" size="small" icon="trash" @click="del(kb.id)">删除</UiButton>
          </div>
          <div class="kb-item-actions">
            <label class="field upload-field"><span class="field-label">术语文件（CSV / JSON / TXT）</span><input type="file" accept=".csv,.json,.txt" @change="e => pickFile(kb.id, e)" /></label>
            <UiButton size="small" icon="upload" @click="upload(kb.id)">上传新版本</UiButton>
            <UiButton variant="secondary" size="small" icon="chevron" :aria-expanded="Boolean(versions[kb.id])" :aria-controls="'versions-' + kb.id" @click="toggleVersions(kb)">{{ versions[kb.id] ? '收起版本' : '查看版本' }}</UiButton>
          </div>
          <div v-if="versions[kb.id]" :id="'versions-' + kb.id" class="version-panel">
            <div v-if="versions[kb.id].length" class="table-scroll">
              <table>
                <thead><tr><th>版本</th><th>文件</th><th>术语数</th><th>大小</th><th>sha256</th><th>创建时间</th><th>操作</th></tr></thead>
                <tbody>
                  <tr v-for="v in versions[kb.id]" :key="v.id">
                    <td><span class="cell-primary numeric">v{{ v.version }}</span> <StatusTag v-if="v.version === kb.latest_version" status="done" plain>最新</StatusTag></td>
                    <td class="cell-secondary">{{ v.file_name }}</td>
                    <td class="numeric">{{ v.term_count }}</td>
                    <td class="numeric">{{ fmtSize(v.size_bytes) }}</td>
                    <td class="cell-mono" :title="v.sha256">{{ (v.sha256 || '').slice(0,12) }}…</td>
                    <td class="cell-secondary version-date">{{ v.created_at }}</td>
                    <td><div class="cell-actions"><UiButton :href="dlVersion(kb, v)" target="_blank" rel="noopener" variant="ghost" size="small">下载</UiButton><UiButton variant="quiet-danger" size="small" :disabled="v.version === kb.latest_version" @click="delVersion(kb, v)">删除</UiButton></div></td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div v-else class="empty-state compact"><p>暂无版本。</p></div>
          </div>
        </article>
      </div>
    </section>
  </div>
</template>

<style scoped>
.library-card { padding: 0; overflow: hidden; }
.library-header { align-items: flex-end; margin: 0; padding: 20px 24px; border-bottom: 1px solid var(--color-divider); }
.search-field { width: min(320px, 100%); }
.kb-directory { display: grid; }
.kb-item + .kb-item { border-top: 1px solid var(--color-divider); }
.kb-item-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 20px 24px 12px; }
.kb-identity { display: flex; min-width: 0; align-items: center; gap: 12px; }
.kb-identity h3 { margin: 0; font-size: 16px; overflow-wrap: anywhere; }
.kb-identity p { margin: 2px 0 0; color: var(--color-text-tertiary); font-size: 12px; }
.kb-icon { display: grid; width: 36px; height: 36px; flex: 0 0 36px; place-items: center; border-radius: var(--radius-md); corner-shape: squircle; background: var(--color-info-bg); color: var(--color-info); }
.kb-item-actions { display: flex; align-items: flex-end; gap: 8px; padding: 0 24px 20px 72px; }
.upload-field { max-width: 420px; flex: 1; }
.version-panel { border-top: 1px solid var(--color-divider); background: color-mix(in srgb, var(--color-sunken) 30%, var(--color-surface)); }
.version-date { min-width: 160px; font-size: 12px; }
.compact { padding: 24px; }
@media (max-width: 700px) {
  .library-header { align-items: stretch; }
  .search-field { width: 100%; }
  .kb-item-header, .kb-item-actions { padding-inline: 16px; }
  .kb-item-actions { align-items: stretch; flex-wrap: wrap; padding-left: 16px; }
  .upload-field { flex-basis: 100%; max-width: none; }
}
</style>
