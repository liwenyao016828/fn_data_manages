<template>
  <span
    class="status-dot"
    :class="[sizeClass, statusClass, blinkClass]"
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

const statusClass = computed(() => {
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

const blinkClass = computed(() => {
  return props.status === 'online' || props.status === 'checking' ? 'blink-dot' : ''
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
