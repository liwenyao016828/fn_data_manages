<template>
  <span
    class="inline-block rounded-full shrink-0"
    :class="[sizeClass, colorClass, blinkClass]"
  />
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  status: {
    type: String,
    default: 'default',
    validator: (v) => ['online', 'offline', 'selected', 'checking', 'default'].includes(v),
  },
  size: {
    type: String,
    default: 'sm',
    validator: (v) => ['xs', 'sm', 'md'].includes(v),
  },
})

const sizeClass = computed(() => {
  const map = { xs: 'w-1.5 h-1.5', sm: 'w-2 h-2', md: 'w-2.5 h-2.5' }
  return map[props.size]
})

const colorClass = computed(() => {
  const map = {
    online: 'bg-emerald-500',
    offline: 'bg-red-500',
    selected: 'bg-[#4facfe]',
    checking: 'bg-amber-400',
    default: 'bg-[#D9D9D9]',
  }
  return map[props.status]
})

const blinkClass = computed(() => {
  return props.status === 'online' || props.status === 'checking' ? 'blink-dot' : ''
})
</script>
