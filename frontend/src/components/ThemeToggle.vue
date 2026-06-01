<template>
  <button
    type="button"
    class="theme-toggle group relative inline-flex h-7 w-[52px] shrink-0 cursor-pointer items-center rounded-full border outline-none transition-colors duration-300 focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
    :class="isDark
      ? 'bg-[var(--soft-info-bg)] border-[color-mix(in_srgb,var(--text-info)_30%,transparent)]'
      : 'bg-[var(--emphasis-bg)] border-[color-mix(in_srgb,var(--text-warning)_35%,transparent)]'"
    role="switch"
    :aria-checked="isDark"
    :aria-label="isDark ? '切换到白天模式' : '切换到夜间模式'"
    @click="toggleTheme"
  >
    <span
      class="toggle-thumb pointer-events-none absolute top-[3px] flex h-[20px] w-[20px] items-center justify-center rounded-full shadow-md transition-all duration-300 ease-out"
      :class="isDark
        ? 'translate-x-[27px] bg-[var(--card)] text-[var(--icon-info)]'
        : 'translate-x-[3px] bg-[var(--card)] text-[var(--text-warning)]'"
    >
      <Transition name="icon-swap" mode="out-in">
        <component :is="isDark ? Moon : Sun" :key="isDark ? 'moon' : 'sun'" :size="12" :stroke-width="2.25" class="icon-swap-in" />
      </Transition>
    </span>
    <span class="toggle-track-icons pointer-events-none absolute inset-0 flex items-center justify-between px-[7px] text-[10px]">
      <Sun :size="9" :stroke-width="2" class="transition-opacity duration-200" :class="isDark ? 'opacity-30 text-[var(--icon-warning)]' : 'opacity-0'" />
      <Moon :size="9" :stroke-width="2" class="transition-opacity duration-200" :class="isDark ? 'opacity-0' : 'opacity-30 text-[var(--icon-info)]'" />
    </span>
  </button>
</template>

<script setup>
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useThemeStore } from '../stores/theme'
import { Sun, Moon } from 'lucide-vue-next'

const themeStore = useThemeStore()
const { theme } = storeToRefs(themeStore)
const isDark = computed(() => theme.value === 'dark')

const toggleTheme = () => {
  themeStore.toggleTheme()
}
</script>

<style scoped>
.icon-swap-enter-active,
.icon-swap-leave-active {
  transition: transform 0.25s ease, opacity 0.25s ease;
}
.icon-swap-enter-from {
  transform: rotate(-90deg) scale(0.6);
  opacity: 0;
}
.icon-swap-leave-to {
  transform: rotate(90deg) scale(0.6);
  opacity: 0;
}
</style>
