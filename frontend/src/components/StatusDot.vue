<template>
  <span
    class="inline-block rounded-full shrink-0 transition-all duration-200"
    :class="[sizeClass, statusColorClass, glowClass, blinkClass]"
    :style="glowStyle"
    :aria-label="ariaLabel"
  />
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  status: {
    type: String,
    default: 'default',
    validator: (v) => ['online', 'offline', 'selected', 'checking', 'warning', 'info', 'default'].includes(v),
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

const statusColorClass = computed(() => {
  const map = {
    online: 'status-dot-online',
    offline: 'status-dot-offline',
    selected: 'status-dot-selected',
    checking: 'status-dot-warning',
    warning: 'status-dot-warning',
    info: 'status-dot-info',
    default: 'status-dot-default',
  }
  return map[props.status] || 'status-dot-default'
})

const glowClass = computed(() => {
  return ['online', 'selected'].includes(props.status) ? 'status-dot' : 'status-dot'
})

const glowStyle = computed(() => {
  if (props.status === 'online') {
    return { boxShadow: '0 0 6px color-mix(in srgb, var(--success) 50%, transparent)' }
  }
  if (props.status === 'selected') {
    return { boxShadow: '0 0 6px color-mix(in srgb, var(--accent) 50%, transparent)' }
  }
  return {}
})

const blinkClass = computed(() => {
  return props.status === 'checking' ? 'blink-dot' : ''
})

const ariaLabel = computed(() => {
  const map = {
    online: '在线',
    offline: '离线',
    selected: '已选中',
    checking: '检测中',
    warning: '警告',
    info: '提示',
    default: '默认',
  }
  return map[props.status] || '默认'
})
</script>
