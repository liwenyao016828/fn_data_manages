<template>
  <button
    type="button"
    class="relative inline-flex h-8 w-[52px] shrink-0 cursor-pointer items-center rounded-full transition-colors duration-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
    :style="{ background: 'var(--muted)' }"
    role="switch"
    :aria-checked="isDark"
    :aria-label="isDark ? '切换到白天模式' : '切换到夜间模式'"
    @click="toggleTheme"
  >
    <Sun
      class="absolute left-[7px] h-3.5 w-3.5 transition-colors duration-300"
      :style="{ color: !isDark ? 'var(--warning)' : 'var(--text-tertiary)' }"
      :stroke-width="2"
    />
    <Moon
      class="absolute right-[7px] h-3.5 w-3.5 transition-colors duration-300"
      :style="{ color: isDark ? 'var(--accent)' : 'var(--text-tertiary)' }"
      :stroke-width="2"
    />
    <span
      class="pointer-events-none block h-6 w-6 rounded-full shadow-[0_1px_3px_rgba(0,0,0,0.15)] ring-0 transition-transform duration-300"
      :class="isDark ? 'translate-x-[24px]' : 'translate-x-[2px]'"
      :style="{ background: '#ffffff' }"
    />
  </button>
</template>

<script setup>
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useThemeStore } from '../stores/theme'
import { Sun, Moon } from 'lucide-vue-next'

const themeStore = useThemeStore()
const { theme, themeMode } = storeToRefs(themeStore)
const isDark = computed(() => theme.value === 'dark')

const toggleTheme = () => {
  themeStore.toggleTheme()
}
</script>
