<script setup>
import { ref, onMounted } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { api } from '../api.js'

const router = useRouter()
const tasks = ref([])
const err = ref('')
async function load() { try { tasks.value = await api.tasks() || [] } catch (e) { err.value = e.message } }
async function del(id) { if (!confirm('删除任务？')) return; try { await api.deleteTask(id); await load() } catch (e) { err.value = e.message } }
function tagClass(s){ return s }
onMounted(() => { load(); setInterval(load, 4000) })
</script>

<template>
  <div class="row" style="justify-content:space-between">
    <h2>任务</h2>
    <button @click="router.push('/tasks/new')">＋ 新建任务</button>
  </div>
  <div v-if="err" class="card" style="color:#b91c1c">{{ err }}</div>
  <table>
    <thead><tr><th>任务</th><th>视频</th><th>状态</th><th>进度</th><th>创建时间</th><th></th></tr></thead>
    <tbody>
      <tr v-for="t in tasks" :key="t.id">
        <td style="font-family:monospace;font-size:12px">{{ t.id }}</td>
        <td>{{ t.filename }}</td>
        <td><span class="tag" :class="t.status">{{ t.status }}</span></td>
        <td>{{ t.progress }}%</td>
        <td>{{ t.created_at }}</td>
        <td>
          <RouterLink :to="'/tasks/' + t.id"><button class="ghost">详情</button></RouterLink>
          <button class="danger" style="margin-left:6px" @click="del(t.id)">删除</button>
        </td>
      </tr>
      <tr v-if="!tasks.length"><td colspan="6" style="color:#94a3b8">暂无任务</td></tr>
    </tbody>
  </table>
</template>
