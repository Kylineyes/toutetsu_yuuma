<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api.js'

const router = useRouter()
const password = ref('')
const err = ref('')
async function submit() {
  err.value = ''
  try {
    await api.login(password.value)
    router.push('/tasks')
  } catch (e) { err.value = e.message }
}
</script>

<template>
  <div style="max-width:360px;margin:12vh auto">
    <div class="card">
      <h2>登录 · toutetsu_yuuma</h2>
      <p style="color:#64748b">输入访问口令（即服务端环境变量 PASSWORD）。若服务端未设口令，直接进入即可。</p>
      <div class="row">
        <input v-model="password" type="password" placeholder="访问口令" style="flex:1" @keyup.enter="submit" />
        <button @click="submit">登录</button>
      </div>
      <p v-if="err" style="color:#b91c1c">{{ err }}</p>
      <p><a href="/tasks">不登录，直接进入（若未启用口令）</a></p>
    </div>
  </div>
</template>
