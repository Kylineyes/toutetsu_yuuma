<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { RouterLink } from 'vue-router'
import { api } from '../api.js'
import PageHeader from '../components/PageHeader.vue'
import StatusTag from '../components/StatusTag.vue'
import UiButton from '../components/UiButton.vue'

const props = defineProps({ id: { type: String, required: true } })
const task = ref(null)
const stages = ref([])
const result = ref(null)
const err = ref('')
const msg = ref('')
let timer = null

async function load() {
  try {
    task.value = await api.task(props.id)
    stages.value = await api.taskStages(props.id) || []
    if (['done', 'failed'].includes(task.value.status)) {
      result.value = await api.taskResult(props.id)
      if (timer) { clearInterval(timer); timer = null }
    }
  } catch (e) { err.value = e.message }
}
function start() { if (!timer) timer = setInterval(load, 3000) }
async function retry() {
  try { await api.retryTask(props.id); result.value = null; stages.value = []; await load(); start() } catch (e) { err.value = e.message }
}
async function cleanup() {
  if (!confirm('删除该任务的临时中间文件（tmp 缓存）？')) return
  try { await api.cleanupCache(props.id); msg.value = '已清理 tmp 缓存' } catch (e) { err.value = e.message }
}
function dl(name) { return api.fileUrl(props.id, name) }
const cost = computed(() => result.value?.cost || {})
onMounted(() => { load(); start() })
onUnmounted(() => { if (timer) clearInterval(timer) })
</script>

<template>
  <div class="page">
    <PageHeader :title="'任务详情'" :eyebrow="'任务 / ' + id" :description="task ? task.filename : '正在读取任务信息。'">
      <template #actions>
        <UiButton to="/tasks" variant="ghost" icon="arrow-left">返回列表</UiButton>
        <UiButton v-if="task?.status === 'failed'" variant="secondary" icon="refresh" @click="retry">重跑任务</UiButton>
        <UiButton v-if="['done','failed','paused'].includes(task?.status)" variant="secondary" icon="trash" @click="cleanup">清理缓存</UiButton>
      </template>
    </PageHeader>
    <div v-if="err" class="alert" role="alert"><p>{{ err }}</p></div>
    <div v-if="msg" class="alert alert-success" role="status"><p>{{ msg }}</p></div>

    <template v-if="task">
      <section class="card overview-card" aria-labelledby="overview-title">
        <div class="card-header">
          <div><h2 id="overview-title" class="card-title">运行概览</h2><p class="card-description cell-mono">{{ task.id }}</p></div>
          <StatusTag :status="task.status" />
        </div>
        <div class="progress-summary"><strong>{{ task.progress }}%</strong><span class="muted">总体进度</span></div>
        <div class="progress-track" role="progressbar" aria-label="任务总体进度" aria-valuemin="0" aria-valuemax="100" :aria-valuenow="task.progress">
          <div class="progress-value" :style="{ width: task.progress + '%' }" />
        </div>
        <dl class="meta-list task-meta">
          <div><dt>视频</dt><dd>{{ task.filename }}</dd></div>
          <div><dt>阅读深度</dt><dd class="numeric">T={{ task.reading_depth }}</dd></div>
          <div><dt>截图间隔</dt><dd class="numeric">{{ task.screenshot_interval }}s</dd></div>
          <div><dt>开始</dt><dd>{{ task.started_at || '—' }}</dd></div>
          <div><dt>结束</dt><dd>{{ task.finished_at || '—' }}</dd></div>
        </dl>
        <p v-if="task.error" class="task-error">{{ task.error }}</p>
      </section>

      <section class="card card-flush" aria-labelledby="stages-title">
        <div class="card-header"><div><h2 id="stages-title" class="card-title">处理阶段</h2><p class="card-description">阶段可并行；运行中的任务每 3 秒自动刷新。</p></div></div>
        <div v-if="stages.length" class="table-scroll">
          <table>
            <thead><tr><th>阶段</th><th>状态</th><th>进度</th><th>耗时</th><th>备注</th></tr></thead>
            <tbody>
              <tr v-for="s in stages" :key="s.id">
                <td class="cell-primary cell-mono">{{ s.stage }}</td>
                <td><StatusTag :status="s.status" /></td>
                <td class="numeric">{{ s.progress }}%</td>
                <td class="numeric">{{ s.duration_ms ? (s.duration_ms/1000).toFixed(1) + 's' : '—' }}</td>
                <td class="cell-secondary stage-message">{{ s.message || s.error || '—' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="empty-state compact"><p>阶段尚未开始。</p></div>
      </section>

      <section v-if="result" class="card" aria-labelledby="artifacts-title">
        <div class="card-header"><div><h2 id="artifacts-title" class="card-title">产物与成本</h2><p class="card-description">下载任务产物，核对本次处理的资源消耗。</p></div></div>
        <dl class="cost-grid">
          <div><dt>ASR 时长</dt><dd>{{ cost.asr_seconds || 0 }}<small>s</small></dd></div>
          <div><dt>识图数量</dt><dd>{{ cost.vision_images || 0 }}<small>张</small></dd></div>
          <div><dt>输入 Tokens</dt><dd>{{ cost.prompt_tokens || 0 }}</dd></div>
          <div><dt>输出 Tokens</dt><dd>{{ cost.completion_tokens || 0 }}</dd></div>
        </dl>
        <div class="artifact-list">
          <UiButton v-for="file in [
            { name: 'summary.md', label: '视频总结' },
            { name: 'knowledge.md', label: '知识卡片' },
            { name: 'fused.srt', label: '统一字幕' },
            { name: 'asr.srt', label: 'ASR 字幕' },
            { name: 'vision.srt', label: '识图字幕' },
          ]" :key="file.name" variant="secondary" icon="download" :href="dl(file.name)" target="_blank" rel="noopener">{{ file.label }}</UiButton>
        </div>
      </section>

      <section v-if="result?.summary_md" class="card" aria-labelledby="summary-title">
        <div class="card-header"><h2 id="summary-title" class="card-title">视频总结</h2><span class="section-number">summary.md</span></div>
        <pre class="md">{{ result.summary_md }}</pre>
      </section>
      <section v-if="result?.knowledge_md" class="card" aria-labelledby="knowledge-title">
        <div class="card-header"><h2 id="knowledge-title" class="card-title">知识卡片</h2><span class="section-number">knowledge.md</span></div>
        <pre class="md">{{ result.knowledge_md }}</pre>
      </section>
    </template>
  </div>
</template>

<style scoped>
.progress-summary { display: flex; align-items: baseline; gap: 10px; margin-bottom: 8px; }
.progress-summary strong { font-family: var(--font-mono); font-size: 24px; font-variant-numeric: tabular-nums; }
.progress-summary span { font-size: 12px; }
.task-meta { margin-top: 20px; }
.task-meta > div { max-width: 100%; }
.task-meta dd { margin-left: 6px; overflow-wrap: anywhere; }
.task-error { margin: 20px 0 0; padding-top: 16px; border-top: 1px solid var(--color-divider); color: var(--color-danger); }
.stage-message { min-width: 180px; max-width: 420px; overflow-wrap: anywhere; }
.compact { padding: 24px; }
.cost-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; margin: 0 0 24px; }
.cost-grid > div { padding: 12px 16px; border-radius: var(--radius-md); corner-shape: squircle; background: var(--color-sunken); }
.cost-grid dt { color: var(--color-text-secondary); font-size: 12px; }
.cost-grid dd { margin: 4px 0 0; font-family: var(--font-mono); font-size: 20px; font-variant-numeric: tabular-nums; }
.cost-grid small { margin-left: 4px; color: var(--color-text-tertiary); font-family: var(--font-sans); font-size: 12px; }
.artifact-list { display: flex; flex-wrap: wrap; gap: 8px; }
.artifact-list .button { min-width: 160px; flex: 1; }
@media (max-width: 700px) { .cost-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
</style>
