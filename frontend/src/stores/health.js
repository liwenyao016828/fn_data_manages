import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { healthApi } from '../api/health'

export const useHealthStore = defineStore('health', () => {
  const statusMap = ref({})
  const detailsMap = ref({})
  const polling = ref(false)
  let pollingTimer = null

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
    statusMap.value = { ...statusMap.value, ...map }
    detailsMap.value = { ...detailsMap.value, ...details }
  }

  const refreshAll = async () => {
    try {
      const res = await healthApi.getAll()
      if (res.data.code === 0 && res.data.data && res.data.data.length > 0) {
        processResults(res.data.data)
      }
    } catch (e) {
      console.error('Health refresh failed:', e)
    }
  }

  const checkOne = async (uid) => {
    try {
      const res = await healthApi.getOne(uid)
      if (res.data.code === 0 && res.data.data) {
        statusMap.value = { ...statusMap.value, [uid]: res.data.data.online }
        detailsMap.value = { ...detailsMap.value, [uid]: res.data.data }
      }
    } catch (e) {
      console.error('Check one failed:', e)
    }
  }

  const forceCheckAll = async () => {
    try {
      const res = await healthApi.forceCheckAll()
      if (res.data.code === 0 && res.data.data) {
        processResults(res.data.data)
      }
    } catch (e) {
      console.error('Force health check failed:', e)
    }
  }

  const forceCheckOne = async (uid) => {
    try {
      const res = await healthApi.forceCheck(uid)
      if (res.data.code === 0 && res.data.data) {
        statusMap.value = { ...statusMap.value, [uid]: res.data.data.online }
        detailsMap.value = { ...detailsMap.value, [uid]: res.data.data }
      }
    } catch (e) {
      console.error('Force check one failed:', e)
    }
  }

  const setStatus = (uid, online) => {
    statusMap.value = { ...statusMap.value, [uid]: online }
  }

  const startPolling = (intervalMs = 15000) => {
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
    startPolling,
    stopPolling,
  }
})
