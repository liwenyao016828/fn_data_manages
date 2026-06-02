<template>
  <button
    type="button"
    class="theme-toggle relative inline-flex h-7 w-12 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
    :class="isDark ? 'bg-zinc-700' : 'bg-slate-200'"
    role="switch"
    :aria-checked="isDark"
    :aria-label="isDark ? '切换到白天模式' : '切换到夜间模式'"
    @click="toggleTheme"
  >
    <span
      class="toggle-thumb pointer-events-none block h-5 w-5 rounded-full shadow-lg ring-0 transition-transform duration-200 flex items-center justify-center"
      :class="isDark ? 'translate-x-5 bg-white' : 'translate-x-0 bg-white'"
    >
      <Transition name="icon-swap" mode="out-in">
        <Sun v-if="!isDark" :key="'sun'" class="h-3 w-3 text-amber-500" :stroke-width="2.25" />
        <Moon v-else :key="'moon'" class="h-3 w-3 text-slate-700" :stroke-width="2.25" />
      </Transition>
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
  transition: transform 0.2s ease, opacity 0.2s ease;
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
