<script setup>
import { ref, computed, onMounted, onUnmounted, watch, provide } from 'vue'
import axios from 'axios'
import { storeToRefs } from 'pinia'
import { useAppContext } from './stores/context'
import { useHealthStore } from './stores/health'
import { instanceUid } from '@/lib/instance'
import { LayoutDashboard, MonitorCog, HardDrive, Shield, FileText, Settings, PanelLeftClose, PanelLeftOpen } from 'lucide-vue-next'
import { Button } from '@/components/ui/Button.vue'
import { ScrollArea } from '@/components/ui/ScrollArea.vue'
import { Toaster } from '@/components/ui/Sonner.vue'
import MessageToast from '@/components/ui/MessageToast.vue'
import { useMessage } from './composables/useMessage'
import DataManageView from './components/DataManageView.vue'
import BackupView from './components/BackupView.vue'
import SettingsView from './components/SettingsView.vue'
import DashboardView from './components/DashboardView.vue'
import ManagementView from './components/ManagementView.vue'
import LogsView from './components/LogsView.vue'
import StatusDot from './components/StatusDot.vue'

const store = useAppContext()
const healthStore = useHealthStore()
const { message } = useMessage()
const { current, isActive, userName, host, port, name, logEnabled } = storeToRefs(store)
const { favorites } = storeToRefs(store)
const { statusMap: onlineStatus } = storeToRefs(healthStore)

const activeMenu = ref('dashboard')
const navRequest = ref(null)
const showSwitcher = ref(false)
const sidebarCollapsed = ref(false)
const compactMode = ref(localStorage.getItem('compactMode') === 'true')
const instances = ref([])
const systemInfo = ref({ username: '', hostname: '', os: '' })
const progressWidth = ref(0)
const progressVisible = ref(false)

provide('sidebarCollapsed', sidebarCollapsed)

onMounted(() => {
  window.addEventListener('toggle-sidebar', () => {
    sidebarCollapsed.value = !sidebarCollapsed.value
  })
  window.addEventListener('compact-mode-change', (e) => {
    compactMode.value = e.detail
  })
  healthStore.startPolling(15000)
})

onUnmounted(() => {
  healthStore.stopPolling()
})

const avatarText = computed(() => {
  if (isActive.value && name.value) {
    const matches = name.value.match(/[A-Z]/g)
    if (matches && matches.length >= 2) return matches.slice(0, 2).join('')
    if (matches && matches.length === 1) return matches[0] + (name.value.replace(/[^a-zA-Z]/g, '').charAt(matches[0].length) || '').toUpperCase()
    return name.value.slice(0, 2).toUpperCase()
  }
  if (systemInfo.value.username) {
    const matches = systemInfo.value.username.match(/[A-Z]/g)
    if (matches && matches.length >= 2) return matches.slice(0, 2).join('')
    return systemInfo.value.username.slice(0, 2).toUpperCase()
  }
  return 'DB'
})

const navItems = computed(() => {
  const items = [
    { key: 'dashboard', label: '控制台', icon: LayoutDashboard },
    { key: 'management', label: '连接管理', icon: MonitorCog },
    { key: 'data', label: '数据管理', icon: HardDrive },
    { key: 'backup', label: '备份管理', icon: Shield },
  ]
  if (logEnabled.value) {
    items.push({ key: 'logs', label: '日志中心', icon: FileText })
  }
  items.push({ key: 'settings', label: '设置', icon: Settings })
  return items
})

const handleMenuSelect = (key) => {
  activeMenu.value = key
  startProgress()
}

let progressTimer1 = null
let progressTimer2 = null
let progressTimer3 = null
let progressAutoCompleteTimer = null
const startProgress = () => {
  progressVisible.value = true
  progressWidth.value = 0
  clearTimeout(progressTimer1)
  clearTimeout(progressTimer2)
  clearTimeout(progressTimer3)
  clearTimeout(progressAutoCompleteTimer)
  progressWidth.value = 15
  progressTimer1 = setTimeout(() => { progressWidth.value = 35 }, 200)
  progressTimer2 = setTimeout(() => { progressWidth.value = 60 }, 500)
  progressTimer3 = setTimeout(() => { progressWidth.value = 80 }, 1000)
  progressAutoCompleteTimer = setTimeout(() => {
    progressWidth.value = 100
    setTimeout(() => {
      progressVisible.value = false
      progressWidth.value = 0
    }, 300)
  }, 3000)
}

const completeProgress = () => {
  clearTimeout(progressAutoCompleteTimer)
  progressWidth.value = 100
  progressTimer1 = setTimeout(() => {
    progressVisible.value = false
    progressWidth.value = 0
  }, 300)
}

const handleNavigate = (payload) => {
  const { action, row } = payload
  if (action === 'logs' && !logEnabled.value) return
  if (action === 'data' || action === 'backup' || action === 'logs') {
    const inst = {
      connectionId: instanceUid(row),
      userName: row.username,
      dbName: row.database || '',
      type: row.type,
      host: row.host,
      port: row.port,
      isRemote: row.isRemote || false,
      name: row.name,
    }
    store.setContext(inst)
    navRequest.value = { ...row, _ts: Date.now() }
    activeMenu.value = action
    startProgress()
  }
}

provide('completeProgress', completeProgress)

const clearNavRequest = () => {
  navRequest.value = null
}

const checkOnlineStatus = (inst) => {
  let uid
  if (inst && inst.id) {
    uid = instanceUid(inst)
  } else if (inst && inst.connectionId) {
    uid = inst.connectionId
  } else {
    return
  }
  healthStore.checkOne(uid)
}

const openSwitcher = async () => {
  instances.value = await store.loadInstances()
  showSwitcher.value = true
}

const selectContext = (inst) => {
  store.setContext({
    connectionId: instanceUid(inst),
    userName: inst.username,
    dbName: '',
    type: inst.type,
    host: inst.host,
    port: inst.port,
    isRemote: inst.isRemote || false,
    name: inst.name,
  })
  showSwitcher.value = false
}

const closeSwitcher = () => {
  showSwitcher.value = false
}

watch(logEnabled, (val) => {
  if (!val && activeMenu.value === 'logs') {
    activeMenu.value = 'dashboard'
  }
})

watch(current, (val) => {
  if (val?.connectionId) {
    checkOnlineStatus(val)
  }
}, { immediate: true, deep: true })

onMounted(async () => {
    try {
    const { data } = await axios.get('/api/system/info')
    systemInfo.value = data
  } catch (e) { console.error('Failed to load system info:', e) }
  store.loadFavorites()
  if (current.value && current.value.connectionId) {
    checkOnlineStatus(current.value)
    return
  }
  if (store.favorites.length > 0) {
    const fav = store.favorites[0]
    const all = await store.loadInstances()
    const found = all.find(i => instanceUid(i) === fav.id)
    if (found) {
      store.setContext({
        connectionId: instanceUid(found),
        userName: found.username,
        dbName: '',
        type: found.type,
        host: found.host,
        port: found.port,
        isRemote: found.isRemote || false,
        name: found.name,
      })
    }
  }
})
</script>

<template>
  <div class="flex h-dvh w-full bg-background" :class="{ compact: compactMode }">
    <aside
      class="sticky top-0 z-30 flex h-dvh shrink-0 flex-col bg-sidebar transition-all duration-300 ease-in-out max-md:w-[56px]"
      :class="sidebarCollapsed ? 'w-[56px]' : 'w-[200px]'"
    >
      <div class="flex h-[52px] items-center shrink-0" :class="sidebarCollapsed ? 'justify-center px-0' : 'gap-2.5 px-3'">
        <div
          class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-[#fff]/10 cursor-pointer hover:bg-[#fff]/15 transition-colors duration-150"
          @click="sidebarCollapsed = !sidebarCollapsed"
        >
          <PanelLeftClose v-if="!sidebarCollapsed" class="h-3.5 w-3.5 text-white/70" />
          <PanelLeftOpen v-else class="h-3.5 w-3.5 text-white/70" />
        </div>
        <span v-if="!sidebarCollapsed" class="truncate text-[13px] font-semibold text-white tracking-tight">
          数据库管理
        </span>
      </div>

      <ScrollArea class="flex-1">
        <nav class="flex flex-col gap-[2px] px-2 py-2">
          <Button
            v-for="item in navItems"
            :key="item.key"
            :variant="activeMenu === item.key ? 'default' : 'ghost'"
            :class="[
              'group w-full rounded-md h-[34px] text-[13px] transition-all duration-150',
              sidebarCollapsed ? 'justify-center px-0' : 'justify-start gap-3 px-3',
              activeMenu === item.key
                ? 'bg-sidebar-accent text-sidebar-accent-foreground shadow-none hover:bg-sidebar-accent'
                : 'text-sidebar-foreground/50 hover:text-sidebar-foreground/80 hover:bg-sidebar-accent/60'
            ]"
            @click="handleMenuSelect(item.key)"
          >
            <component
              :is="item.icon"
              :class="[
                'h-4 w-4 shrink-0 transition-colors duration-150',
                activeMenu === item.key ? 'text-white' : 'text-white/40 group-hover:text-white/70'
              ]"
            />
            <span v-if="!sidebarCollapsed" class="truncate">{{ item.label }}</span>
          </Button>
        </nav>
      </ScrollArea>

      <div class="px-2 pb-3 relative">
        <div
          class="flex items-center gap-2 rounded-md cursor-pointer transition-colors duration-150 hover:bg-sidebar-accent/70"
          :class="[
            isActive ? 'bg-sidebar-accent/40' : 'bg-sidebar-accent/20',
            sidebarCollapsed ? 'justify-center px-1 py-2' : 'px-3 py-2'
          ]"
          @click="openSwitcher"
        >
          <div class="h-6 w-6 rounded-full flex items-center justify-center shrink-0"
            :class="isActive ? 'bg-white/15' : 'bg-white/10'"
          >
            <span class="text-[9px] font-bold tracking-tight" :class="isActive ? 'text-white' : 'text-white/60'">{{ avatarText }}</span>
          </div>
          <div v-if="!sidebarCollapsed" class="flex-1 min-w-0">
            <div class="text-[12px] text-white/70 truncate flex items-center gap-1.5">
              <StatusDot v-if="isActive" :status="onlineStatus[current?.connectionId] === undefined ? 'checking' : (onlineStatus[current?.connectionId] !== false ? 'online' : 'offline')" size="xs" />
              {{ isActive ? (userName || name) : (systemInfo.username || '系统用户') }}
            </div>
            <div class="text-[10px] text-white/30 truncate">
              {{ isActive ? `${host}:${port}` : (systemInfo.hostname || '') }}
            </div>
          </div>
        </div>

        <div
          v-if="showSwitcher"
          class="absolute bottom-full mb-1 bg-sidebar border border-sidebar-border rounded-lg shadow-xl overflow-hidden z-50"
          :class="sidebarCollapsed ? 'left-2 right-2' : 'left-3 right-3'"
          @click.stop
        >
          <div class="flex items-center justify-between px-3 py-2 border-b border-white/10">
            <span class="text-[11px] text-white/50">选择数据库</span>
            <Button variant="ghost" size="icon" class="h-5 w-5 text-white/40 hover:text-white/70" @click="closeSwitcher">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </Button>
          </div>
          <div class="max-h-[260px] overflow-y-auto">
            <div
              v-for="inst in instances"
              :key="(inst.isRemote ? 'r' : 'l') + inst.id"
              class="flex items-center gap-2.5 px-3 py-2 cursor-pointer transition-colors hover:bg-white/5"
              :class="current?.connectionId === instanceUid(inst) ? 'bg-sidebar-primary/10' : ''"
              @click="selectContext(inst)"
            >
              <StatusDot :status="current?.connectionId === instanceUid(inst) ? 'selected' : (onlineStatus[instanceUid(inst)] === undefined ? 'checking' : (onlineStatus[instanceUid(inst)] !== false ? 'online' : 'offline'))" size="sm" />
              <div class="flex-1 min-w-0">
                <div class="text-[12px] text-white/80 truncate">{{ inst.name }}</div>
                <div class="text-[10px] text-white/30 truncate">{{ inst.username }}@{{ inst.host }}:{{ inst.port }}</div>
              </div>
              <span v-if="inst.isRemote" class="text-[9px] text-orange-400/70 bg-orange-400/10 px-1 rounded">远程</span>
            </div>
          </div>
        </div>
      </div>
    </aside>

    <main class="flex-1 min-w-0 overflow-hidden bg-background fade-in">
      <KeepAlive :include="['DashboardView', 'DataManageView', 'BackupView', 'ManagementView', 'LogsView', 'SettingsView']">
        <DashboardView v-if="activeMenu === 'dashboard'" />
        <DataManageView
          v-else-if="activeMenu === 'data'"
          :nav-request="navRequest"
          @nav-accepted="clearNavRequest"
        />
        <BackupView
          v-else-if="activeMenu === 'backup'"
          :nav-request="navRequest"
          @nav-accepted="clearNavRequest"
        />
        <ManagementView v-else-if="activeMenu === 'management'" @navigate="handleNavigate" />
        <LogsView v-else-if="activeMenu === 'logs'" :nav-request="navRequest" @nav-accepted="clearNavRequest" />
        <SettingsView v-else-if="activeMenu === 'settings'" />
      </KeepAlive>
    </main>

    <div
      v-if="progressVisible"
      class="fixed top-0 left-0 right-0 h-[3px] z-[9999]"
    >
      <div
        class="h-full bg-primary transition-all duration-300 ease-out"
        :style="{ width: progressWidth + '%' }"
      />
    </div>

    <Toaster position="top-center" :duration="4000" />
    <MessageToast :message="message" />

    <div v-if="showSwitcher" class="fixed inset-0 z-20" @click="closeSwitcher" />
  </div>
</template>
