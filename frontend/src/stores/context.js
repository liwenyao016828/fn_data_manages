import { ref, computed, watch } from 'vue'
import { defineStore } from 'pinia'
import axios from 'axios'
import { databaseApi } from '../api/database'
import { sourceValue } from '@/lib/instance'
import { STORAGE_KEYS, safeStorage } from '../lib/storageKeys'

const FAVORITES_KEY = STORAGE_KEYS.CONTEXT_FAVORITES
const CURRENT_CONTEXT_KEY = STORAGE_KEYS.CONTEXT_CURRENT

const loadLogEnabled = () => {
  try {
    const raw = safeStorage.get(STORAGE_KEYS.LOG_CONFIG)
    if (raw) {
      const cfg = JSON.parse(raw)
      return cfg.enabled !== false
    }
  } catch (e) { console.error(e) }
  return true
}

const syncLogEnabledFromBackend = async (logEnabledRef) => {
  try {
    const res = await fetch('/api/log-config')
    const data = await res.json()
    if (data.code === 0 && data.data && data.data.enabled !== undefined) {
      logEnabledRef.value = data.data.enabled !== false
    }
  } catch (e) { console.error(e) }
}

const loadPersistedContext = () => {
  try {
    const raw = safeStorage.get(CURRENT_CONTEXT_KEY)
    if (raw) return JSON.parse(raw)
  } catch (e) { console.error(e) }
  return null
}

export const useAppContext = defineStore('context', () => {
  const current = ref(loadPersistedContext())
  const favorites = ref([])
  const logEnabled = ref(loadLogEnabled())

  syncLogEnabledFromBackend(logEnabled)

  watch(logEnabled, (val) => {
    try {
      const raw = safeStorage.get(STORAGE_KEYS.LOG_CONFIG)
      const existing = raw ? JSON.parse(raw) : {}
      safeStorage.set(STORAGE_KEYS.LOG_CONFIG, JSON.stringify({ ...existing, enabled: val }))
    } catch (e) {
      console.error(e)
    }
  }, { immediate: false })

  const isActive = computed(() => current.value !== null)
  const connectionId = computed(() => current.value?.connectionId ?? null)
  const serverId = computed(() => {
    const uid = current.value?.connectionId
    if (!uid || typeof uid !== 'string') return uid
    const parts = uid.split(':')
    return parts.length === 2 ? parseInt(parts[1]) : uid
  })
  const userName = computed(() => current.value?.userName ?? '')
  const dbName = computed(() => current.value?.dbName ?? '')
  const dbType = computed(() => current.value?.type ?? '')
  const host = computed(() => current.value?.host ?? '')
  const port = computed(() => current.value?.port ?? 0)
  const isRemote = computed(() => current.value?.isRemote ?? false)
  const name = computed(() => current.value?.name ?? '')
  const isRedis = computed(() => current.value?.type === 'redis')
  const isPostgreSQL = computed(() => current.value?.type === 'postgresql')
  const isSQLite = computed(() => current.value?.type === 'sqlite')
  const isMariaDB = computed(() => current.value?.type === 'mariadb')
  const isSQLLike = computed(() => ['mysql', 'mariadb', 'postgresql', 'sqlite'].includes(current.value?.type))
  const hostname = ref('')
  const osName = ref('')

  const fetchSystemInfo = async () => {
    try {
      const res = await fetch('/api/system/info')
      const data = await res.json()
      if (data.hostname) hostname.value = data.hostname
      if (data.os) osName.value = data.os
    } catch (e) { console.error(e) }
  }
  fetchSystemInfo()

  const setContext = (ctx) => {
    if (ctx) {
      addFavorite(ctx.connectionId, ctx.name)
    }
    current.value = ctx ? { ...ctx } : null
    if (current.value) {
      safeStorage.set(CURRENT_CONTEXT_KEY, JSON.stringify(current.value))
    } else {
      safeStorage.remove(CURRENT_CONTEXT_KEY)
    }
  }

  const clearContext = () => {
    current.value = null
    safeStorage.remove(CURRENT_CONTEXT_KEY)
  }

  const setDatabase = (dbName) => {
    if (current.value) {
      current.value = { ...current.value, dbName }
      safeStorage.set(CURRENT_CONTEXT_KEY, JSON.stringify(current.value))
    }
  }

  const addFavorite = (id, name) => {
    const existing = favorites.value.find(f => f.id === id)
    if (existing) {
      existing.count = (existing.count || 0) + 1
      existing.lastUsed = Date.now()
      favorites.value.sort((a, b) => (b.count || 0) - (a.count || 0))
    } else {
      favorites.value.unshift({ id, name, count: 1, lastUsed: Date.now() })
      if (favorites.value.length > 5) favorites.value.pop()
    }
    safeStorage.set(FAVORITES_KEY, JSON.stringify(favorites.value))
  }

  const loadFavorites = () => {
    try {
      const raw = safeStorage.get(FAVORITES_KEY)
      if (raw) favorites.value = JSON.parse(raw)
    } catch (e) { console.error(e) }
  }

  const loadInstances = async () => {
    const [local, remote] = await Promise.all([
      databaseApi.list('all').then(res => {
        if (res.data.code === 0) return res.data.data || []
        return []
      }).catch(() => []),
      axios.get('/api/remote-servers').then(res => {
        if (res.data.code === 0) {
          return (res.data.data || []).map(srv => ({
            ...srv, isRemote: true
          }))
        }
        return []
      }).catch(() => [])
    ])
    return [...local, ...remote]
  }

  const setLogEnabled = (val) => {
    logEnabled.value = val
    let fullConfig = { enabled: val }
    try {
      const raw = safeStorage.get(STORAGE_KEYS.LOG_CONFIG)
      const existing = raw ? JSON.parse(raw) : {}
      fullConfig = { ...existing, enabled: val }
      safeStorage.set(STORAGE_KEYS.LOG_CONFIG, JSON.stringify(fullConfig))
    } catch (e) { console.error(e) }
    fetch('/api/log-config', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(fullConfig)
    }).catch(e => console.error(e))
  }

  return {
    current,
    isActive,
    connectionId,
    serverId,
    userName,
    dbName,
    dbType,
    host,
    port,
    isRemote,
    name,
    isRedis,
    isPostgreSQL,
    isSQLite,
    isMariaDB,
    isSQLLike,
    logEnabled,
    favorites,
    hostname,
    osName,
    setContext,
    clearContext,
    setDatabase,
    addFavorite,
    loadFavorites,
    loadInstances,
    setLogEnabled,
  }
})