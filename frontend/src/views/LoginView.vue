<script setup>
import { ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { api } from '../api.js'
import UiButton from '../components/UiButton.vue'

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
  <main class="login-page">
    <section class="login-shell" aria-labelledby="login-title">
      <div class="login-brand">
        <span class="brand-mark" aria-hidden="true">TY</span>
        <div>
          <p class="brand-name">toutetsu_yuuma</p>
          <p class="muted">视频阅读工作台</p>
        </div>
      </div>
      <div class="card login-card">
        <p class="eyebrow">欢迎回来</p>
        <h1 id="login-title">登录工作台</h1>
        <p class="muted">输入服务端访问口令，继续管理视频阅读任务。</p>
        <div v-if="err" class="alert" role="alert"><p>{{ err }}</p></div>
        <form class="login-form" @submit.prevent="submit">
          <label class="field">
            <span class="field-label">访问口令</span>
            <input v-model="password" type="password" autocomplete="current-password" placeholder="输入 PASSWORD" autofocus />
          </label>
          <UiButton type="submit">登录</UiButton>
        </form>
      </div>
      <p class="login-footer">
        服务端未启用口令？<RouterLink to="/tasks">直接进入工作台</RouterLink>
      </p>
    </section>
  </main>
</template>

<style scoped>
.login-brand .muted { margin: 2px 0 0; font-size: 12px; }
.login-card .alert { margin: 16px 0 0; }
</style>
