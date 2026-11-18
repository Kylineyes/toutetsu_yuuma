<script setup>
defineProps({ name: { type: String, required: true } })
</script>

<template>
  <svg class="icon" viewBox="0 0 24 24" aria-hidden="true">
    <template v-if="name === 'tasks'"><rect x="5" y="4" width="14" height="16" rx="2"/><path d="M9 4v16M12 9h4M12 13h4"/></template>
    <template v-else-if="name === 'plus'"><path d="M12 5v14M5 12h14"/></template>
    <template v-else-if="name === 'profile'"><path d="M4 7h10M18 7h2M4 17h2M10 17h10"/><circle cx="16" cy="7" r="2"/><circle cx="8" cy="17" r="2"/></template>
    <template v-else-if="name === 'knowledge'"><path d="M4 5.5A2.5 2.5 0 0 1 6.5 3H11v16H6.5A2.5 2.5 0 0 0 4 21.5zM20 5.5A2.5 2.5 0 0 0 17.5 3H13v16h4.5a2.5 2.5 0 0 1 2.5 2.5z"/></template>
    <template v-else-if="name === 'sun'"><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.42 1.42M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.42-1.42M17.66 6.34l1.41-1.41"/></template>
    <template v-else-if="name === 'moon'"><path d="M20.5 14.2A8.5 8.5 0 0 1 9.8 3.5 8.5 8.5 0 1 0 20.5 14.2z"/></template>
    <template v-else-if="name === 'logout'"><path d="M10 5H5v14h5M14 8l4 4-4 4M8 12h10"/></template>
    <template v-else-if="name === 'arrow-left'"><path d="M19 12H5M11 6l-6 6 6 6"/></template>
    <template v-else-if="name === 'arrow-right'"><path d="M5 12h14M13 6l6 6-6 6"/></template>
    <template v-else-if="name === 'trash'"><path d="M4 7h16M9 7V4h6v3M7 7l1 13h8l1-13M10 11v5M14 11v5"/></template>
    <template v-else-if="name === 'refresh'"><path d="M20 6v5h-5M4 18v-5h5M6.1 8a7 7 0 0 1 11.7-2L20 11M4 13l2.2 5a7 7 0 0 0 11.7-2"/></template>
    <template v-else-if="name === 'download'"><path d="M12 3v12M7 10l5 5 5-5M5 20h14"/></template>
    <template v-else-if="name === 'search'"><circle cx="10.5" cy="10.5" r="6.5"/><path d="m16 16 4 4"/></template>
    <template v-else-if="name === 'upload'"><path d="M12 16V4M7 9l5-5 5 5M5 20h14"/></template>
    <template v-else-if="name === 'chevron'"><path d="m8 10 4 4 4-4"/></template>
    <template v-else-if="name === 'file'"><path d="M6 3h8l4 4v14H6zM14 3v5h4M9 13h6M9 17h4"/></template>
    <template v-else-if="name === 'check'"><path d="m5 12 4 4L19 6"/></template>
    <template v-else><circle cx="12" cy="12" r="8"/></template>
  </svg>
</template>
