import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import LoginView from './views/LoginView.vue'
import TasksView from './views/TasksView.vue'
import TaskCreate from './views/TaskCreate.vue'
import TaskDetail from './views/TaskDetail.vue'
import ProfilesView from './views/ProfilesView.vue'
import KnowledgeBasesView from './views/KnowledgeBasesView.vue'
import { api } from './api.js'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/tasks' },
    { path: '/login', component: LoginView },
    { path: '/tasks', component: TasksView },
    { path: '/tasks/new', component: TaskCreate },
    { path: '/tasks/:id', component: TaskDetail, props: true },
    { path: '/profiles', component: ProfilesView },
    { path: '/knowledge-bases', component: KnowledgeBasesView },
  ],
})

router.beforeEach(async (to) => {
  if (to.path === '/login') return true
  try {
    const s = await api.session()
    if (s.login_required && !s.authenticated) return '/login'
  } catch (e) { /* ignore */ }
  return true
})

createApp(App).use(router).mount('#app')
