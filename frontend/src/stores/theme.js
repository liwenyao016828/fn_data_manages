import { ref, watch, computed } from 'vue'
import { defineStore } from 'pinia'
import { STORAGE_KEYS, safeStorage } from '../lib/storageKeys'

const getSystemTheme = () => {
  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }
  return 'light'
}

const resolveInitialTheme = () => {
  try {
    const mode = safeStorage.get(STORAGE_KEYS.THEME_MODE)
    if (mode === 'light' || mode === 'dark') {
      return mode
    }
    if (mode === 'auto') {
      return getSystemTheme()
    }
    const saved = safeStorage.get(STORAGE_KEYS.THEME)
    if (saved === 'dark' || saved === 'light') return saved
  } catch (e) { console.error(e) }
  return getSystemTheme()
}

export const useThemeStore = defineStore('theme', () => {
  const initialMode = (() => {
    const m = safeStorage.get(STORAGE_KEYS.THEME_MODE)
    if (m === 'auto' || m === 'light' || m === 'dark') return m
    return 'auto'
  })()
  const theme = ref(resolveInitialTheme())
  const themeMode = ref(initialMode)

  const isDark = computed(() => theme.value === 'dark')

  const effectiveTheme = computed(() => {
    if (themeMode.value === 'auto') {
      return getSystemTheme()
    }
    return theme.value
  })

  const setTheme = (val) => {
    theme.value = val
    safeStorage.set(STORAGE_KEYS.THEME, val)
    applyTheme(val)
  }

  const setThemeMode = (mode) => {
    themeMode.value = mode
    safeStorage.set(STORAGE_KEYS.THEME_MODE, mode)
    if (mode === 'auto') {
      theme.value = getSystemTheme()
    } else {
      theme.value = mode
    }
  }

  const toggleTheme = () => {
    const next = theme.value === 'dark' ? 'light' : 'dark'
    setTheme(next)
    safeStorage.set(STORAGE_KEYS.THEME_MODE, next)
    themeMode.value = next
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
      if (themeMode.value === 'auto') {
        const next = e.matches ? 'dark' : 'light'
        theme.value = next
        applyTheme(next)
      }
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
    themeMode,
    isDark,
    effectiveTheme,
    setTheme,
    setThemeMode,
    toggleTheme,
  }
})
