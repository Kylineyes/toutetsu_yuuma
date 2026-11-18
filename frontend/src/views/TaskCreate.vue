<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api.js'
import PageHeader from '../components/PageHeader.vue'
import UiButton from '../components/UiButton.vue'

const router = useRouter()
const err = ref('')
const caps = ['summary', 'asr', 'vision', 'reading']
const capLabel = { summary: '总结', asr: 'ASR 语音识别', vision: '识图', reading: '阅读理解' }
const profiles = ref({ summary: [], asr: [], vision: [], reading: [] })
const kbs = ref([])
const versionsMap = ref({})
const selectedKBs = ref({})
const kbOrder = ref([])
const form = ref({
  sourceType: 'file', video_url: '',
  summary_profile_id: '', asr_profile_id: '', vision_profile_id: '', reading_comprehension_profile_id: '',
  generate_ai_subtitle_via_asr: false, generate_vision_subtitle: false,
  screenshot_mode: 'simple', screenshot_interval: 60, reading_depth: 2,
  summary_preference: '',
})
const files = ref({ video: null, native: null, ai: null, vision: null, danmu: null })

async function load() {
  // 后端无数据时返回 null；逐项隔离失败，保留其余可用配置。
  for (const c of caps) {
    try {
      const list = await api.profiles(c)
      profiles.value[c] = Array.isArray(list) ? list : []
    } catch (e) { profiles.value[c] = [] }
  }
  try {
    const list = await api.kbs()
    kbs.value = Array.isArray(list) ? list : []
  } catch (e) { kbs.value = [] }
}
async function toggleKB(kb) {
  if (selectedKBs.value[kb.id] !== undefined) {
    delete selectedKBs.value[kb.id]
    delete versionsMap.value[kb.id]
    kbOrder.value = kbOrder.value.filter(id => id !== String(kb.id))
    return
  }
  if (!kb.latest_version) { err.value = '该知识库尚无版本，无法选择'; return }
  let vs = []
  try { vs = (await api.kbVersions(kb.id)) || [] } catch (e) { vs = [] }
  if (!Array.isArray(vs)) vs = []
  versionsMap.value[kb.id] = vs
  selectedKBs.value[kb.id] = vs.length ? vs[0].version : kb.latest_version
  kbOrder.value.push(String(kb.id))
}
function kbPriority(kbId) {
  const i = kbOrder.value.indexOf(String(kbId))
  return i < 0 ? 0 : i + 1
}
function submit() {
  err.value = ''
  const fd = new FormData()
  if (form.value.sourceType === 'url') {
    if (!form.value.video_url) { err.value = '请填视频直链'; return }
    fd.append('video_url', form.value.video_url)
  } else {
    if (!files.value.video) { err.value = '请选择视频文件'; return }
    fd.append('video_file', files.value.video)
  }
  const si = Number(form.value.screenshot_interval)
  if (!Number.isInteger(si) || si <= 0) { err.value = '截图间隔须为正整数秒'; return }
  const rd = Number(form.value.reading_depth)
  if (!Number.isInteger(rd) || rd < 0) { err.value = '阅读深度须为非负整数'; return }
  for (const k of ['native', 'ai', 'vision', 'danmu']) if (files.value[k]) fd.append('subtitle_' + k, files.value[k])
  fd.append('summary_profile_id', form.value.summary_profile_id || 0)
  fd.append('asr_profile_id', form.value.asr_profile_id || 0)
  fd.append('vision_profile_id', form.value.vision_profile_id || 0)
  fd.append('reading_comprehension_profile_id', form.value.reading_comprehension_profile_id || 0)
  fd.append('generate_ai_subtitle_via_asr', form.value.generate_ai_subtitle_via_asr ? '1' : '0')
  fd.append('generate_vision_subtitle', form.value.generate_vision_subtitle ? '1' : '0')
  fd.append('screenshot_mode', form.value.screenshot_mode)
  fd.append('screenshot_interval', form.value.screenshot_interval)
  fd.append('reading_depth', form.value.reading_depth)
  fd.append('summary_preference', form.value.summary_preference || '')
  for (const kid of kbOrder.value) fd.append('kb', kid + ':' + selectedKBs.value[kid])
  api.createTask(fd).then((t) => router.push('/tasks/' + t.id)).catch((e) => { err.value = e.message })
}
onMounted(load)
</script>

<template>
  <div class="page create-page">
    <PageHeader title="新建任务" eyebrow="任务配置" description="导入视频与字幕，选择处理能力，生成总结、知识卡和统一字幕。" />
    <div v-if="err" class="alert" role="alert"><p>{{ err }}</p></div>
    <form @submit.prevent="submit">
      <section class="card">
        <div class="card-header">
          <div><h2 class="card-title">视频源</h2><p class="card-description">上传本地文件，或提供可直接访问的视频地址。</p></div>
          <span class="section-number">01</span>
        </div>
        <div class="choice-row source-switch" role="group" aria-label="视频来源">
          <label class="choice"><input v-model="form.sourceType" type="radio" value="file" />上传文件</label>
          <label class="choice"><input v-model="form.sourceType" type="radio" value="url" />直链 URL</label>
        </div>
        <label v-if="form.sourceType === 'file'" class="field">
          <span class="field-label">视频文件</span>
          <input type="file" accept="video/*" @change="e => files.video = e.target.files[0]" />
        </label>
        <label v-else class="field">
          <span class="field-label">视频直链</span>
          <input v-model="form.video_url" type="url" placeholder="https://.../video.mp4" />
        </label>
      </section>

      <section class="card">
        <div class="card-header">
          <div><h2 class="card-title">字幕与弹幕</h2><p class="card-description">每个来源最多一份，均可留空；也可交由服务生成。</p></div>
          <span class="section-number">02</span>
        </div>
        <div class="form-grid">
          <label class="field"><span class="field-label">原生字幕</span><input type="file" @change="e => files.native = e.target.files[0]" /></label>
          <label class="field"><span class="field-label">AI 字幕</span><input type="file" @change="e => files.ai = e.target.files[0]" /></label>
          <label class="field"><span class="field-label">识图字幕</span><input type="file" @change="e => files.vision = e.target.files[0]" /></label>
          <label class="field"><span class="field-label">弹幕列表</span><input type="file" @change="e => files.danmu = e.target.files[0]" /></label>
        </div>
        <div class="choice-row generation-options">
          <label class="choice"><input v-model="form.generate_ai_subtitle_via_asr" type="checkbox" />由 ASR 生成 AI 字幕</label>
          <label class="choice"><input v-model="form.generate_vision_subtitle" type="checkbox" />由识图模型生成字幕</label>
        </div>
      </section>

      <section class="card">
        <div class="card-header">
          <div><h2 class="card-title">配置方案</h2><p class="card-description">四类能力独立选择，也可暂不指定。</p></div>
          <span class="section-number">03</span>
        </div>
        <div class="form-grid">
          <label v-for="c in caps" :key="c" class="field">
            <span class="field-label">{{ capLabel[c] }}</span>
            <select v-model="form[c === 'reading' ? 'reading_comprehension_profile_id' : c + '_profile_id']">
              <option value="">不指定方案</option>
              <option v-for="p in profiles[c]" :key="p.id" :value="p.id">#{{ p.id }} · {{ p.name }} · {{ p.model }}</option>
            </select>
          </label>
        </div>
      </section>

      <section class="card">
        <div class="card-header">
          <div><h2 class="card-title">解析参数</h2><p class="card-description">控制画面采样密度与总结阶段的回看深度。</p></div>
          <span class="section-number">04</span>
        </div>
        <div class="form-grid form-grid-three">
          <label class="field"><span class="field-label">截图模式</span><select v-model="form.screenshot_mode"><option value="simple">simple（简易）</option></select></label>
          <label class="field"><span class="field-label">截图间隔（秒）</span><input v-model.number="form.screenshot_interval" type="number" min="1" step="1" /></label>
          <label class="field"><span class="field-label">阅读深度 T</span><input v-model.number="form.reading_depth" type="number" min="0" step="1" /></label>
          <label class="field field-span"><span class="field-label">总结方向偏好（可空）</span><textarea v-model="form.summary_preference" rows="3" placeholder="说明希望重点关注的主题、术语或内容结构" /></label>
        </div>
      </section>

      <section class="card">
        <div class="card-header">
          <div><h2 class="card-title">知识库</h2><p class="card-description">可多选并指定版本；点击顺序即注入优先级。</p></div>
          <span class="section-number">05</span>
        </div>
        <div v-if="kbs.length" class="kb-list">
          <div v-for="kb in kbs" :key="kb.id" class="kb-option" :class="{ selected: selectedKBs[kb.id] !== undefined }">
            <label class="kb-choice">
              <input type="checkbox" :checked="selectedKBs[kb.id] !== undefined" @change="toggleKB(kb)" />
              <span class="cell-primary">{{ kb.name }}</span>
              <span class="tertiary">最新 v{{ kb.latest_version }}</span>
            </label>
            <span v-if="selectedKBs[kb.id] !== undefined" class="tag tag-running tag-plain">优先级 {{ kbPriority(kb.id) }}</span>
            <select v-if="selectedKBs[kb.id] !== undefined" v-model.number="selectedKBs[kb.id]" class="kb-version" aria-label="选择知识库版本">
              <option v-for="v in (versionsMap[kb.id] || [])" :key="v.id" :value="v.version">v{{ v.version }} · {{ v.term_count }} 词</option>
            </select>
          </div>
        </div>
        <div v-else class="empty-state compact"><p>暂无可用知识库，可跳过此步骤。</p></div>
      </section>

      <div class="submit-bar">
        <p class="field-hint">提交后可在任务详情查看阶段进度与产物。</p>
        <UiButton type="submit" icon="arrow-right">提交任务</UiButton>
      </div>
    </form>
  </div>
</template>

<style scoped>
.create-page { max-width: 960px; }
.source-switch { margin-bottom: 20px; }
.generation-options { margin-top: 20px; }
.kb-list { display: grid; gap: 8px; }
.kb-option { display: flex; min-height: 48px; align-items: center; gap: 12px; padding: 8px 12px; border: 1px solid var(--color-border); border-radius: var(--radius-md); corner-shape: squircle; }
.kb-option.selected { border-color: var(--color-primary); background: color-mix(in srgb, var(--color-info-bg) 45%, var(--color-surface)); }
.kb-choice { display: flex; min-width: 0; flex: 1; align-items: center; gap: 10px; cursor: pointer; }
.kb-choice .tertiary { margin-left: auto; font-size: 12px; }
.kb-version { width: auto; min-width: 150px; }
.submit-bar { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 24px 0 0; }
.submit-bar p { margin: 0; }
.compact { padding: 20px; }
@media (max-width: 640px) {
  .kb-option { align-items: stretch; flex-wrap: wrap; }
  .kb-choice { flex-basis: 100%; }
  .kb-choice .tertiary { margin-left: 0; }
  .kb-version { flex: 1; }
  .submit-bar { align-items: stretch; flex-direction: column; }
}
</style>
