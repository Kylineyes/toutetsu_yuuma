<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import UiIcon from './UiIcon.vue'

const theme = ref(document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light')
const media = window.matchMedia('(prefers-color-scheme: dark)')

function apply(next, persist = false) {
  theme.value = next
  document.documentElement.dataset.theme = next
  if (persist) localStorage.setItem('ty-theme', next)
}
function toggle() { apply(theme.value === 'dark' ? 'light' : 'dark', true) }
function followSystem(event) {
  if (!localStorage.getItem('ty-theme')) apply(event.matches ? 'dark' : 'light')
}
onMounted(() => media.addEventListener('change', followSystem))
onUnmounted(() => media.removeEventListener('change', followSystem))
</script>

<template>
  <button
    class="button button-ghost"
    type="button"
    :aria-label="theme === 'dark' ? '切换到浅色主题' : '切换到深色主题'"
    :aria-pressed="theme === 'dark' ? 'true' : 'false'"
    :title="theme === 'dark' ? '切换到浅色主题' : '切换到深色主题'"
    @click="toggle"
  >
    <UiIcon :name="theme === 'dark' ? 'sun' : 'moon'" />
    <span class="button-label">{{ theme === 'dark' ? '浅色主题' : '深色主题' }}</span>
  </button>
</template>
