<script setup>
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { api } from './api.js'
import UiIcon from './components/UiIcon.vue'
import ThemeToggle from './components/ThemeToggle.vue'

const route = useRoute()
const router = useRouter()
const nav = [
  { to: '/tasks', label: '任务列表', icon: 'tasks' },
  { to: '/tasks/new', label: '新建任务', icon: 'plus' },
  { to: '/profiles', label: '配置方案', icon: 'profile' },
  { to: '/knowledge-bases', label: '知识库', icon: 'knowledge' },
]
async function logout() {
  try { await api.logout() } catch (e) { /* 服务端不可达时仍清理当前页面会话 */ }
  router.push('/login')
}
</script>

<template>
  <div v-if="route.path === '/login'" class="app">
    <RouterView />
    <ThemeToggle class="theme-floating" />
  </div>
  <div v-else class="app">
    <aside class="sidebar">
      <RouterLink class="brand" to="/tasks" aria-label="返回任务列表">
        <span class="brand-mark" aria-hidden="true">TY</span>
        <div>
          <p class="brand-name">toutetsu_yuuma</p>
          <p class="brand-tagline">视频阅读工作台</p>
        </div>
      </RouterLink>
      <p class="nav-label">工作空间</p>
      <nav class="sidebar-nav" aria-label="主导航">
        <RouterLink v-for="item in nav" :key="item.to" class="nav-item" :to="item.to">
          <UiIcon :name="item.icon" />
          <span>{{ item.label }}</span>
        </RouterLink>
      </nav>
      <div class="sidebar-footer">
        <ThemeToggle />
        <button class="button button-ghost" type="button" @click="logout">
          <UiIcon name="logout" /><span class="button-label">退出登录</span>
        </button>
        <p class="sidebar-footer-meta">总结 · 知识卡 · 用语库</p>
      </div>
    </aside>
    <main class="content">
      <div class="content-inner"><RouterView /></div>
    </main>
  </div>
</template>
