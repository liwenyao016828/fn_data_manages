<template>
  <span
    v-if="props.status === 'online'"
    class="inline-block rounded-full shrink-0 transition-all duration-200"
    :class="[
      sizeClass,
      'bg-emerald-500 dark:bg-emerald-400',
      'shadow-[0_0_8px_rgba(16,185,129,0.4)] dark:shadow-[0_0_12px_rgba(16,185,129,0.5)]',
      'animate-pulse',
    ]"
    :aria-label="ariaLabel"
  />
  <span
    v-else-if="props.status === 'offline'"
    class="inline-block rounded-full shrink-0 transition-all duration-200"
    :class="[
      sizeClass,
      'bg-red-500 dark:bg-red-400',
    ]"
    :aria-label="ariaLabel"
  />
  <span
    v-else-if="props.status === 'selected'"
    class="inline-block rounded-full shrink-0 transition-all duration-200"
    :class="[
      sizeClass,
      'bg-blue-600 dark:bg-blue-400',
      'shadow-[0_0_8px_rgba(37,99,235,0.4)] dark:shadow-[0_0_12px_rgba(96,165,250,0.5)]',
    ]"
    :aria-label="ariaLabel"
  />
  <span
    v-else
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
