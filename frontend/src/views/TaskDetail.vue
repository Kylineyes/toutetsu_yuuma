<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { api } from '../api.js'

const props = defineProps({ id: { type: String, required: true } })
const task = ref(null)
const result = ref(null)
const err = ref('')
let timer = null

async function load() {
  try {
    task.value = await api.task(props.id)
    if (task.value.status === 'done' || task.value.status === 'failed') {
      result.value = await api.taskResult(props.id)
      if (timer) { clearInterval(timer); timer = null }
    }
  } catch (e) { err.value = e.message }
}
async function retry() {
  try { await api.retryTask(props.id); result.value = null; await load(); start() } catch (e) { err.value = e.message }
}
async function cleanup() {
  if (!confirm('删除该任务的临时中间文件（tmp 缓存）？')) return
  try { await api.cleanupCache(props.id); msg.value = '已清理 tmp 缓存' } catch (e) { err.value = e.message }
}
const msg = ref('')
function start() { if (!timer) timer = setInterval(load, 3000) }
function dl(name) { return api.fileUrl(props.id, name) }
const cost = computed(() => result.value?.cost || {})
const subs = computed(() => result.value?.subtitles || [])
onMounted(() => { load(); start() })
onUnmounted(() => { if (timer) clearInterval(timer) })
</script>

<template>
  <div v-if="err" class="card" style="color:#b91c1c">{{ err }}</div>
  <div v-if="msg" class="card" style="color:#166534">{{ msg }}</div>
  <template v-if="task">
    <div class="row" style="justify-content:space-between">
      <h2>任务 {{ task.id }}</h2>
      <div>
        <button v-if="task.status==='failed'" class="ghost" @click="retry">重跑</button>
        <button v-if="['done','failed','paused'].includes(task.status)" class="ghost" style="margin-left:6px" @click="cleanup">清理缓存</button>
      </div>
    </div>
    <div class="card">
      <div class="row">
        <span class="tag" :class="task.status">{{ task.status }}</span>
        <span>阶段：{{ task.stage }}</span>
        <span>进度：{{ task.progress }}%</span>
      </div>
      <div style="height:8px;background:#e2e8f0;border-radius:4px;overflow:hidden">
        <div :style="{ width: task.progress + '%', height:'100%', background:'#2563eb' }"></div>
      </div>
      <p v-if="task.error" style="color:#b91c1c">{{ task.error }}</p>
      <p style="color:#64748b;font-size:13px">视频：{{ task.filename }} ｜ 阅读深度 T={{ task.reading_depth }} ｜ 截图间隔 {{ task.screenshot_interval }}s</p>
    </div>

    <div v-if="result" class="card">
      <div class="row">
        <strong>估算成本</strong>
        <span class="tag">ASR {{ cost.asr_seconds || 0 }}s</span>
        <span class="tag">识图 {{ cost.vision_images || 0 }} 张</span>
        <span class="tag">prompt {{ cost.prompt_tokens || 0 }} tok</span>
        <span class="tag">completion {{ cost.completion_tokens || 0 }} tok</span>
      </div>
      <div class="row">
        <a :href="dl('summary.md')" target="_blank"><button class="ghost">下载 summary.md</button></a>
        <a :href="dl('knowledge.md')" target="_blank"><button class="ghost">下载 knowledge.md</button></a>
        <a :href="dl('fused.srt')" target="_blank"><button class="ghost">下载 统一字幕 fused.srt</button></a>
        <a :href="dl('asr.srt')" target="_blank"><button class="ghost">下载 ASR 字幕</button></a>
        <a :href="dl('vision.srt')" target="_blank"><button class="ghost">下载 识图字幕</button></a>
      </div>
    </div>

    <div v-if="result && result.summary_md" class="card">
      <h3>summary.md</h3>
      <pre class="md">{{ result.summary_md }}</pre>
    </div>
    <div v-if="result && result.knowledge_md" class="card">
      <h3>knowledge.md</h3>
      <pre class="md">{{ result.knowledge_md }}</pre>
    </div>
  </template>
</template>
