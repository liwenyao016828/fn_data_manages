import { ref, watch } from 'vue'
import { defineStore } from 'pinia'

const THEME_KEY = 'db_manager_theme'

const resolveInitialTheme = () => {
  try {
    const saved = localStorage.getItem(THEME_KEY)
    if (saved === 'dark' || saved === 'light') return saved
  } catch (e) { console.error(e) }
  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }
  return 'light'
}

export const useThemeStore = defineStore('theme', () => {
  const theme = ref(resolveInitialTheme())

  const isDark = () => theme.value === 'dark'

  const setTheme = (val) => {
    theme.value = val
    try {
      localStorage.setItem(THEME_KEY, val)
    } catch (e) { console.error(e) }
    applyTheme(val)
  }

  const toggleTheme = () => {
    const next = theme.value === 'dark' ? 'light' : 'dark'
    setTheme(next)
  }

  const applyTheme = (val) => {
    if (typeof document === 'undefined') return
    const root = document.documentElement
    if (val === 'dark') {
      root.classList.add('dark')
      root.style.colorScheme = 'dark'
    } else {
      root.classList.remove('dark')
      root.style.colorScheme = 'light'
    }
  }

  const initSystemListener = () => {
    if (typeof window === 'undefined' || !window.matchMedia) return
    const mql = window.matchMedia('(prefers-color-scheme: dark)')
    const handler = (e) => {
      try {
        if (localStorage.getItem(THEME_KEY)) return
      } catch (err) { /* noop */ }
      setTheme(e.matches ? 'dark' : 'light')
    }
    mql.addEventListener?.('change', handler)
  }

  watch(theme, (val) => {
    applyTheme(val)
  }, { immediate: true })

  if (typeof window !== 'undefined') {
    initSystemListener()
  }

  return {
    theme,
    isDark,
    setTheme,
    toggleTheme,
  }
})
