import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import TasksView from './views/TasksView.vue'
import TaskCreate from './views/TaskCreate.vue'
import TaskDetail from './views/TaskDetail.vue'
import ProfilesView from './views/ProfilesView.vue'
import KnowledgeBasesView from './views/KnowledgeBasesView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/tasks' },
    { path: '/tasks', component: TasksView },
    { path: '/tasks/new', component: TaskCreate },
    { path: '/tasks/:id', component: TaskDetail, props: true },
    { path: '/profiles', component: ProfilesView },
    { path: '/knowledge-bases', component: KnowledgeBasesView },
  ],
})

createApp(App).use(router).mount('#app')
