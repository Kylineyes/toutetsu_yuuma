<script setup>
import { computed } from 'vue'

const props = defineProps({ status: { type: String, default: 'pending' }, plain: { type: Boolean, default: false } })
const labels = { pending: '等待中', running: '运行中', done: '已完成', failed: '失败', skipped: '已跳过', paused: '已暂停' }
const known = ['pending', 'running', 'done', 'failed', 'skipped']
const classes = computed(() => ({ [`tag-${props.status}`]: known.includes(props.status), 'tag-plain': props.plain }))
</script>

<template>
  <span class="tag" :class="classes"><slot>{{ labels[status] || status }}</slot></span>
</template>
