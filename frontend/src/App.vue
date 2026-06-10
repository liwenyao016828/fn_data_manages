<script setup>
import { ref, computed, onMounted, onUnmounted, watch, provide } from 'vue'
import axios from 'axios'
import { storeToRefs } from 'pinia'
import { useAppContext } from './stores/context'
import { useHealthStore } from './stores/health'
import { instanceUid } from '@/lib/instance'
import { LayoutDashboard, MonitorCog, HardDrive, Shield, FileText, Settings, PanelLeftClose, PanelLeftOpen, ChevronUp, Database } from 'lucide-vue-next'
import ThemeToggle from './components/ThemeToggle.vue'
import { Button } from '@/components/ui/Button.vue'
import { ScrollArea } from '@/components/ui/ScrollArea.vue'
import { Toaster } from '@/components/ui/Sonner.vue'
import MessageToast from '@/components/ui/MessageToast.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import { useMessage } from './composables/useMessage'
import { STORAGE_KEYS, safeStorage } from './lib/storageKeys'
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
const compactMode = ref(safeStorage.get(STORAGE_KEYS.UI_COMPACT_MODE) === 'true')
const instances = ref([])
const systemInfo = ref({ username: '', hostname: '', os: '' })
const progressWidth = ref(0)
const progressVisible = ref(false)

provide('sidebarCollapsed', sidebarCollapsed)

onMounted(() => {
  window.addEventListener('toggle-sidebar', sidebarToggleHandler)
  window.addEventListener('compact-mode-change', compactModeHandler)
  // #16 从 localStorage 读取健康轮询间隔，缺省 15s
  // 范围 5s-5min；非法值回退到 15s
  const stored = safeStorage.get(STORAGE_KEYS.HEALTH_POLL_INTERVAL_MS)
  const parsed = stored ? parseInt(stored, 10) : NaN
  const healthInterval = Number.isFinite(parsed) && parsed >= 5000 && parsed <= 300000
    ? parsed
    : 15000
  healthStore.startPolling(healthInterval)
})

onUnmounted(() => {
  window.removeEventListener('toggle-sidebar', sidebarToggleHandler)
  window.removeEventListener('compact-mode-change', compactModeHandler)
  healthStore.stopPolling()
})

const sidebarToggleHandler = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
}
const compactModeHandler = (e) => {
  compactMode.value = e.detail
}

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
  if (showSwitcher.value) {
    showSwitcher.value = false
    return
  }
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
    <!-- Sidebar -->
    <aside
      class="sticky top-0 z-30 flex h-dvh shrink-0 flex-col transition-all duration-300 max-md:w-[60px]"
      :class="[sidebarCollapsed ? 'w-[60px]' : 'w-[220px]']"
      style="background: var(--surface); border-right: 1px solid var(--border-subtle);"
    >
      <!-- Logo & Collapse Toggle -->
      <div class="flex h-[56px] items-center shrink-0 px-3" :class="sidebarCollapsed ? 'justify-center' : 'gap-3'">
        <div
          class="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl cursor-pointer transition-all duration-200 hover:bg-muted"
          @click="sidebarCollapsed = !sidebarCollapsed"
          :aria-label="sidebarCollapsed ? '展开侧边栏' : '折叠侧边栏'"
        >
          <PanelLeftClose v-if="!sidebarCollapsed" class="h-4 w-4 icon-secondary" />
          <PanelLeftOpen v-else class="h-4 w-4 icon-secondary" />
        </div>
        <Transition name="fade">
          <div v-if="!sidebarCollapsed" class="flex items-center gap-2 min-w-0">
            <div class="flex h-6 w-6 items-center justify-center rounded-lg" style="background: var(--accent);">
              <Database class="h-3.5 w-3.5 text-white" />
            </div>
            <span class="truncate text-[13px] font-semibold tracking-tight" style="color: var(--text-primary);">数据库管理</span>
          </div>
        </Transition>
      </div>

      <!-- Navigation -->
      <ScrollArea class="flex-1">
        <nav class="flex flex-col gap-0.5 py-1 pl-3.5 pr-2.5">
          <button
            v-for="item in navItems"
            :key="item.key"
            class="group relative w-full rounded-xl text-[13px] transition-all duration-200 cursor-pointer flex items-center"
            :class="[
              sidebarCollapsed ? 'h-9 justify-center' : 'h-9 justify-start gap-3 px-3',
              activeMenu === item.key
                ? 'font-medium'
                : ''
            ]"
            @click="handleMenuSelect(item.key)"
          >
            <!-- Active indicator bar -->
            <div
              v-if="activeMenu === item.key"
              class="absolute -left-3.5 top-1/2 -translate-y-1/2 w-[3px] h-4 rounded-r-full transition-all duration-200"
              style="background: var(--accent);"
            />
            <!-- Active background -->
            <div
              v-if="activeMenu === item.key"
              class="absolute inset-0 rounded-xl transition-colors duration-200"
              style="background: var(--accent-soft);"
            />
            <component
              :is="item.icon"
              :class="[
                'h-[18px] w-[18px] shrink-0 transition-colors duration-200 relative z-10',
                activeMenu === item.key ? 'icon-accent' : 'icon-muted group-hover:icon-secondary'
              ]"
            />
            <span
              v-if="!sidebarCollapsed"
              class="truncate relative z-10 transition-colors duration-200"
              :class="activeMenu === item.key ? 'text-accent' : 'text-sidebar-foreground/60 group-hover:text-sidebar-foreground'"
            >{{ item.label }}</span>
          </button>
        </nav>
      </ScrollArea>

      <!-- Bottom: Theme Toggle + Context Switcher -->
      <div class="px-2.5 pb-3 relative">
        <div
          class="flex items-center rounded-xl cursor-pointer transition-all duration-200 hover:bg-muted"
          :class="sidebarCollapsed ? 'justify-center px-0 py-2.5' : 'gap-3 px-3 py-2.5'"
          @click="openSwitcher"
        >
          <div
            class="h-8 w-8 rounded-xl flex items-center justify-center shrink-0 transition-colors duration-200"
            :style="{ background: isActive ? 'var(--accent)' : 'var(--muted)' }"
          >
            <span class="text-[10px] font-bold tracking-tight" :style="{ color: isActive ? 'var(--accent-foreground)' : 'var(--text-tertiary)' }">{{ avatarText }}</span>
          </div>
          <div v-if="!sidebarCollapsed" class="flex-1 min-w-0">
            <div class="text-[12px] font-medium truncate flex items-center gap-1.5" style="color: var(--text-primary);">
              <StatusDot v-if="isActive" :status="onlineStatus[current?.connectionId] === undefined ? 'checking' : (onlineStatus[current?.connectionId] !== false ? 'online' : 'offline')" size="xs" />
              {{ isActive ? (userName || name) : (systemInfo.username || '系统用户') }}
            </div>
            <div class="text-[11px] truncate" style="color: var(--text-tertiary);">
              {{ isActive ? `${host}:${port}` : (systemInfo.hostname || '') }}
            </div>
          </div>
          <div v-if="!sidebarCollapsed" class="flex items-center gap-1.5 shrink-0" @click.stop>
            <ThemeToggle />
            <ChevronUp class="h-3.5 w-3.5 icon-muted" />
          </div>
        </div>

        <!-- Context Switcher Popup -->
        <Transition name="popup">
          <div
            v-if="showSwitcher"
            class="absolute bottom-full mb-2 rounded-2xl overflow-hidden z-50 border"
            :class="sidebarCollapsed ? 'left-0 right-0' : 'left-0 right-0'"
            style="background: var(--surface-elevated); border-color: var(--border); box-shadow: 0 20px 60px rgba(0,0,0,0.12);"
            @click.stop
          >
            <div class="flex items-center justify-between px-4 py-3" style="border-bottom: 1px solid var(--border-subtle);">
              <span class="text-[12px] font-medium" style="color: var(--text-secondary);">选择数据库</span>
              <button
                class="h-5 w-5 rounded-lg flex items-center justify-center transition-colors duration-200 cursor-pointer hover:bg-muted icon-muted hover:icon-secondary"
                @click="closeSwitcher"
              >
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
              </button>
            </div>
            <div class="max-h-[280px] overflow-y-auto py-1">
              <div
                v-for="inst in instances"
                :key="(inst.isRemote ? 'r' : 'l') + inst.id"
                class="flex items-center gap-3 px-4 py-2.5 cursor-pointer transition-colors duration-150"
                :class="current?.connectionId === instanceUid(inst) ? '' : 'hover:bg-muted'"
                :style="current?.connectionId === instanceUid(inst) ? 'background: var(--accent-soft);' : ''"
                @click="selectContext(inst)"
              >
                <StatusDot :status="current?.connectionId === instanceUid(inst) ? 'selected' : (onlineStatus[instanceUid(inst)] === undefined ? 'checking' : (onlineStatus[instanceUid(inst)] !== false ? 'online' : 'offline'))" size="sm" />
                <div class="flex-1 min-w-0">
                  <div class="text-[12px] font-medium truncate" style="color: var(--text-primary);">{{ inst.name }}</div>
                  <div class="text-[11px] truncate" style="color: var(--text-tertiary);">{{ inst.username }}@{{ inst.host }}:{{ inst.port }}</div>
                </div>
                <span v-if="inst.isRemote" class="badge-status badge-status-warning text-[9px]">远程</span>
              </div>
            </div>
          </div>
        </Transition>
      </div>
    </aside>

    <!-- Main Content -->
    <main class="flex-1 min-w-0 overflow-hidden bg-background">
      <div class="h-full max-w-[1440px] mx-auto">
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
      </div>
    </main>

    <!-- Progress Bar -->
    <Transition name="progress">
      <div
        v-if="progressVisible"
        class="fixed top-0 left-0 right-0 h-[2px] z-[9999]"
      >
        <div
          class="h-full transition-all duration-300 ease-out"
          :style="{ width: progressWidth + '%', background: 'var(--accent)', boxShadow: '0 0 8px color-mix(in srgb, var(--accent) 40%, transparent)' }"
        />
      </div>
    </Transition>

    <Toaster position="top-center" :duration="4000" />
    <MessageToast :message="message" />
    <ConfirmDialog />

    <!-- Switcher Backdrop -->
    <Transition name="fade">
      <div v-if="showSwitcher" class="fixed inset-0 z-20" @click="closeSwitcher" />
    </Transition>
  </div>
</template>

<style scoped>
/* Fade transition */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* Popup transition */
.popup-enter-active {
  transition: all 0.25s cubic-bezier(0.16, 1, 0.3, 1);
}
.popup-leave-active {
  transition: all 0.15s ease-in;
}
.popup-enter-from {
  opacity: 0;
  transform: translateY(8px) scale(0.96);
}
.popup-leave-to {
  opacity: 0;
  transform: translateY(4px) scale(0.98);
}

/* Progress bar transition */
.progress-enter-active {
  transition: opacity 0.15s ease;
}
.progress-leave-active {
  transition: opacity 0.3s ease 0.2s;
}
.progress-enter-from,
.progress-leave-to {
  opacity: 0;
}
</style>
