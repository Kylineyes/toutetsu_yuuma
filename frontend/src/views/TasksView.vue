<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { RouterLink } from 'vue-router'
import { api } from '../api.js'
import PageHeader from '../components/PageHeader.vue'
import StatusTag from '../components/StatusTag.vue'
import UiButton from '../components/UiButton.vue'
import UiIcon from '../components/UiIcon.vue'

const tasks = ref([])
const err = ref('')
let timer = null
async function load() {
  try { tasks.value = await api.tasks() || [] } catch (e) { err.value = e.message }
}
async function del(id) {
  if (!confirm('删除任务？此操作不可撤销。')) return
  try { await api.deleteTask(id); await load() } catch (e) { err.value = e.message }
}
onMounted(() => { load(); timer = setInterval(load, 4000) })
onUnmounted(() => clearInterval(timer))
</script>

<template>
  <div class="page">
    <PageHeader title="任务列表" eyebrow="工作台" description="查看视频阅读进度，进入任务详情或管理历史任务。">
      <template #actions>
        <UiButton to="/tasks/new" icon="plus">新建任务</UiButton>
      </template>
    </PageHeader>
    <div v-if="err" class="alert" role="alert"><p>{{ err }}</p></div>
    <section class="card card-flush" aria-label="任务列表">
      <div class="card-header">
        <div>
          <h2 class="card-title">全部任务</h2>
          <p class="card-description">每 4 秒自动刷新 · 共 {{ tasks.length }} 项</p>
        </div>
      </div>
      <div v-if="tasks.length" class="table-scroll">
        <table>
          <thead><tr><th>任务标识</th><th>视频文件</th><th>状态</th><th>进度</th><th>创建时间</th><th><span class="sr-only">操作</span></th></tr></thead>
          <tbody>
            <tr v-for="t in tasks" :key="t.id">
              <td><RouterLink class="cell-mono task-id" :to="'/tasks/' + t.id">{{ t.id }}</RouterLink></td>
              <td><span class="cell-primary filename">{{ t.filename }}</span></td>
              <td><StatusTag :status="t.status" /></td>
              <td class="numeric progress-cell"><span>{{ t.progress }}%</span><div class="progress-track"><div class="progress-value" :style="{ width: t.progress + '%' }" /></div></td>
              <td class="cell-secondary created-cell">{{ t.created_at }}</td>
              <td>
                <div class="cell-actions">
                  <UiButton :to="'/tasks/' + t.id" variant="ghost" size="small">详情</UiButton>
                  <UiButton variant="quiet-danger" size="small" :aria-label="'删除任务 ' + t.id" @click="del(t.id)">删除</UiButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="empty-state">
        <UiIcon name="file" />
        <p>暂无任务，从第一个视频开始。</p>
        <UiButton to="/tasks/new" variant="secondary" icon="plus">新建任务</UiButton>
      </div>
    </section>
  </div>
</template>

<style scoped>
.task-id { display: inline-block; max-width: 180px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.filename { display: block; min-width: 140px; max-width: 300px; overflow-wrap: anywhere; }
.progress-cell { min-width: 112px; }
.progress-cell .progress-track { width: 96px; height: 4px; margin-top: 6px; }
.created-cell { min-width: 160px; font-size: 12px; }
.empty-state .button { margin-top: 16px; }
</style>
