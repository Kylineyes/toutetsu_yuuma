<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api.js'

const router = useRouter()
const err = ref('')
const caps = ['summary', 'asr', 'vision', 'reading']
const capLabel = { summary: '总结', asr: 'ASR', vision: '识图', reading: '阅读理解' }
const profiles = ref({ summary: [], asr: [], vision: [], reading: [] })
const kbs = ref([])
const versionsMap = ref({})
const selectedKBs = ref({}) // kbId -> version

const form = ref({
  sourceType: 'file', video_url: '',
  summary_profile_id: '', asr_profile_id: '', vision_profile_id: '', reading_comprehension_profile_id: '',
  generate_ai_subtitle_via_asr: false, generate_vision_subtitle: false,
  screenshot_mode: 'simple', screenshot_interval: 60, reading_depth: 2,
  summary_preference: '',
})
const files = ref({ video: null, native: null, ai: null, vision: null, danmu: null })

async function load() {
  for (const c of caps) { try { profiles.value[c] = await api.profiles(c) } catch (e) {} }
  try { kbs.value = await api.kbs() } catch (e) {}
}
async function toggleKB(kb) {
  if (selectedKBs.value[kb.id] !== undefined) { delete selectedKBs.value[kb.id]; return }
  let vs = []
  try { vs = await api.kbVersions(kb.id) } catch (e) {}
  versionsMap.value[kb.id] = vs
  const latest = vs.length ? vs[0].version : kb.latest_version
  selectedKBs.value[kb.id] = latest
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
  for (const [kid, ver] of Object.entries(selectedKBs.value)) fd.append('kb', kid + ':' + ver)
  api.createTask(fd).then((t) => router.push('/tasks/' + t.id)).catch((e) => { err.value = e.message })
}
onMounted(load)
</script>

<template>
  <h2>新建任务</h2>
  <div v-if="err" class="card" style="color:#b91c1c">{{ err }}</div>

  <div class="card">
    <h3>视频源</h3>
    <div class="row">
      <label><input type="radio" value="file" v-model="form.sourceType" /> 上传文件</label>
      <label><input type="radio" value="url" v-model="form.sourceType" /> 直链 URL</label>
    </div>
    <div class="row" v-if="form.sourceType === 'file'">
      <label class="fld">视频文件<input type="file" @change="e => files.video = e.target.files[0]" /></label>
    </div>
    <div class="row" v-else>
      <label class="fld" style="flex:1">视频直链<input v-model="form.video_url" placeholder="https://.../video.mp4" style="width:100%" /></label>
    </div>
  </div>

  <div class="card">
    <h3>字幕（各 1 份，可空）</h3>
    <div class="row">
      <label class="fld">原生字幕<input type="file" @change="e => files.native = e.target.files[0]" /></label>
      <label class="fld">AI 字幕<input type="file" @change="e => files.ai = e.target.files[0]" /></label>
      <label class="fld">识图字幕<input type="file" @change="e => files.vision = e.target.files[0]" /></label>
      <label class="fld">弹幕列表<input type="file" @change="e => files.danmu = e.target.files[0]" /></label>
    </div>
    <div class="row">
      <label><input type="checkbox" v-model="form.generate_ai_subtitle_via_asr" /> AI 字幕由服务用 ASR 生成</label>
      <label><input type="checkbox" v-model="form.generate_vision_subtitle" /> 识图字幕由服务用识图模型生成</label>
    </div>
  </div>

  <div class="card">
    <h3>配置方案（每类选一个）</h3>
    <div class="row">
      <label class="fld" v-for="c in caps" :key="c">{{ capLabel[c] }}
        <select v-model="form[c === 'reading' ? 'reading_comprehension_profile_id' : c + '_profile_id']">
          <option value="">（不选）</option>
          <option v-for="p in profiles[c]" :key="p.id" :value="p.id">#{{ p.id }} {{ p.name }} ({{ p.model }})</option>
        </select>
      </label>
    </div>
  </div>

  <div class="card">
    <h3>解析参数</h3>
    <div class="row">
      <label class="fld">截图模式
        <select v-model="form.screenshot_mode">
          <option value="simple">simple（简易）</option>
        </select>
      </label>
      <label class="fld">截图间隔(秒)<input type="number" v-model.number="form.screenshot_interval" /></label>
      <label class="fld">阅读深度 T<input type="number" v-model.number="form.reading_depth" min="0" /></label>
    </div>
    <label class="fld" style="display:block">总结方向偏好（可空）
      <textarea v-model="form.summary_preference" rows="2" style="width:100%" placeholder="如：重点讲评分标准与酒款背景"></textarea>
    </label>
  </div>

  <div class="card">
    <h3>知识库（可多选，勾选顺序即优先级）</h3>
    <div class="row" v-for="kb in kbs" :key="kb.id">
      <label><input type="checkbox" :checked="selectedKBs[kb.id] !== undefined" @change="toggleKB(kb)" /> {{ kb.name }}（最新 v{{ kb.latest_version }}）</label>
      <select v-if="selectedKBs[kb.id] !== undefined" v-model.number="selectedKBs[kb.id]">
        <option v-for="v in (versionsMap[kb.id] || [])" :key="v.id" :value="v.version">v{{ v.version }}（{{ v.term_count }} 词）</option>
      </select>
    </div>
    <p v-if="!kbs.length" style="color:#94a3b8">暂无知识库</p>
  </div>

  <button @click="submit">提交任务</button>
</template>
