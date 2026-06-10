import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { healthApi } from '../api/health'

export const useHealthStore = defineStore('health', () => {
  const DEFAULT_POLL_INTERVAL_MS = 15000
  const statusMap = ref({})
  const detailsMap = ref({})
  const polling = ref(false)
  let pollingTimer = null

  // 不同类型的请求使用独立的 AbortController，避免互相取消
  let refreshController = null
  let refreshInFlight = false
  let checkOneController = null

  const getStatus = (uid) => {
    return statusMap.value[uid]
  }

  const isOnline = (uid) => {
    return statusMap.value[uid] === true
  }

  const getDetail = (uid) => {
    return detailsMap.value[uid] || null
  }

  const onlineCount = computed(() => {
    return Object.values(statusMap.value).filter(v => v === true).length
  })

  const offlineCount = computed(() => {
    return Object.values(statusMap.value).filter(v => v === false).length
  })

  const totalCount = computed(() => {
    return Object.keys(statusMap.value).length
  })

  const allDetails = computed(() => {
    return Object.values(detailsMap.value)
  })

  const processResults = (items) => {
    const map = {}
    const details = {}
    for (const item of items) {
      map[item.uid] = item.online
      details[item.uid] = item
    }
    statusMap.value = map
    detailsMap.value = details
  }

  // 取消所有正在飞行的请求（仅在 stopPolling / forceCheck 时调用）
  const cancelAll = () => {
    if (refreshController) { try { refreshController.abort() } catch (_) { /* noop */ } }
    refreshController = null
    refreshInFlight = false
    if (checkOneController) { try { checkOneController.abort() } catch (_) { /* noop */ } }
    checkOneController = null
  }

  const refreshAll = async () => {
    // 已有 refreshAll 在飞行中则跳过，避免 abort 导致 ERR_ABORTED
    if (refreshInFlight) return
    refreshInFlight = true
    refreshController = new AbortController()
    try {
      const res = await healthApi.getAll(refreshController.signal)
      if (res?.data?.code === 0 && res.data.data?.length > 0) {
        processResults(res.data.data)
      }
    } catch (e) {
      if (healthApi.isAbort(e)) return
      console.error('Health refresh failed:', e)
    } finally {
      refreshInFlight = false
      refreshController = null
    }
  }

  const checkOne = async (uid) => {
    if (!uid) return
    // 取消上一次 checkOne（独立 controller，不影响 refreshAll）
    if (checkOneController) { try { checkOneController.abort() } catch (_) { /* noop */ } }
    checkOneController = new AbortController()
    try {
      const res = await healthApi.getOne(uid, checkOneController.signal)
      if (res?.data?.code === 0 && res?.data?.data) {
        statusMap.value = { ...statusMap.value, [uid]: res.data.data.online }
        detailsMap.value = { ...detailsMap.value, [uid]: res.data.data }
      }
    } catch (e) {
      if (healthApi.isAbort(e)) return
      console.error('Check one failed:', e)
    }
  }

  const forceCheckAll = async () => {
    cancelAll()
    refreshInFlight = true
    refreshController = new AbortController()
    try {
      const res = await healthApi.forceCheckAll(refreshController.signal)
      if (res?.data?.code === 0 && res?.data?.data) {
        processResults(res.data.data)
      }
    } catch (e) {
      if (healthApi.isAbort(e)) return
      console.error('Force health check failed:', e)
    } finally {
      refreshInFlight = false
      refreshController = null
    }
  }

  const forceCheckOne = async (uid) => {
    if (!uid) return
    cancelAll()
    checkOneController = new AbortController()
    try {
      const res = await healthApi.forceCheck(uid, checkOneController.signal)
      if (res?.data?.code === 0 && res?.data?.data) {
        statusMap.value = { ...statusMap.value, [uid]: res.data.data.online }
        detailsMap.value = { ...detailsMap.value, [uid]: res.data.data }
      }
    } catch (e) {
      if (healthApi.isAbort(e)) return
      console.error('Force check one failed:', e)
    }
  }

  const setStatus = (uid, online) => {
    statusMap.value = { ...statusMap.value, [uid]: online }
  }

  const cleanup = (validUids) => {
    const validSet = new Set(validUids)
    const newStatusMap = {}
    const newDetailsMap = {}
    for (const uid of Object.keys(statusMap.value)) {
      if (validSet.has(uid)) {
        newStatusMap[uid] = statusMap.value[uid]
        newDetailsMap[uid] = detailsMap.value[uid]
      }
    }
    statusMap.value = newStatusMap
    detailsMap.value = newDetailsMap
  }

  const startPolling = (intervalMs = DEFAULT_POLL_INTERVAL_MS) => {
    stopPolling()
    polling.value = true
    refreshAll()
    pollingTimer = setInterval(refreshAll, intervalMs)
  }

  const stopPolling = () => {
    if (pollingTimer) {
      clearInterval(pollingTimer)
      pollingTimer = null
    }
    cancelAll()
    polling.value = false
  }

  return {
    statusMap,
    detailsMap,
    polling,
    getStatus,
    isOnline,
    getDetail,
    onlineCount,
    offlineCount,
    totalCount,
    allDetails,
    refreshAll,
    checkOne,
    forceCheckAll,
    forceCheckOne,
    setStatus,
    cleanup,
    startPolling,
    stopPolling,
  }
})
