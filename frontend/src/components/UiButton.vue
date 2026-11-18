<script setup>
import { RouterLink } from 'vue-router'
import UiIcon from './UiIcon.vue'

defineProps({
  to: { type: [String, Object], default: '' },
  href: { type: String, default: '' },
  variant: { type: String, default: 'primary' },
  size: { type: String, default: 'normal' },
  icon: { type: String, default: '' },
  type: { type: String, default: 'button' },
  disabled: { type: Boolean, default: false },
})
</script>

<template>
  <component
    :is="to ? RouterLink : href ? 'a' : 'button'"
    class="button"
    :class="[`button-${variant}`, { 'button-small': size === 'small' }]"
    :to="to || undefined"
    :href="href || undefined"
    :type="to ? undefined : type"
    :disabled="to ? undefined : disabled"
  >
    <UiIcon v-if="icon" :name="icon" />
    <slot />
  </component>
</template>
