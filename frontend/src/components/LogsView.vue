<template>
  <div class="page-padding h-full flex flex-col overflow-hidden">
    <!-- Page Header -->
    <div class="flex items-center justify-between section-gap shrink-0">
      <div>
        <h2 class="text-[17px] font-semibold text-[var(--text-primary)]">日志中心</h2>
        <p class="text-[13px] text-[var(--text-tertiary)] mt-0.5">查看系统运行日志与数据库日志</p>
      </div>
      <div class="flex items-center gap-2">
        <button class="btn-secondary h-[32px] text-[13px]" @click="activeTab === 'system' ? exportSystemLogs('txt') : exportLogs('txt')">
          <Download class="h-3.5 w-3.5 mr-1.5" />导出
        </button>
        <button v-if="activeTab === 'system' && selectedInst" class="btn-danger h-[32px] text-[13px]" @click="confirmClearSystemLogs">
          <Trash2 class="h-3.5 w-3.5 mr-1.5" />清空
        </button>
      </div>
    </div>

    <div class="content-card flex flex-col flex-1 min-h-0">
      <template v-if="selectedInst">
        <!-- Tab Switcher -->
        <div class="flex items-center justify-between px-5 pt-4 pb-0 shrink-0">
          <div class="inline-flex items-center gap-1 p-1 rounded-xl bg-[var(--muted)]">
            <button
              :class="activeTab === 'system' ? 'tab-active px-4 py-1.5 text-[13px]' : 'tab-inactive px-4 py-1.5 text-[13px]'"
              @click="activeTab = 'system'"
            >系统日志
            </button>
            <button
              v-if="isSqlType(selectedInst.type) || selectedInst.type === 'redis'"
              :class="activeTab === 'database' ? 'tab-active px-4 py-1.5 text-[13px]' : 'tab-inactive px-4 py-1.5 text-[13px]'"
              @click="activeTab = 'database'"
            >数据库日志
            </button>
          </div>
        </div>

        <!-- Instance Selector (database tab) -->
        <div v-if="activeTab === 'database'" class="px-5 pt-3 pb-0 shrink-0">
          <div class="flex gap-1.5 overflow-x-auto pb-2 scrollbar-none">
            <button
              v-for="inst in allInstances"
              :key="(inst.isRemote ? 'r:' : 'l:') + inst.id"
              :class="selectedInstId === instanceUid(inst) ? 'pill pill-active' : 'pill pill-default'"
              @click="selectInstance(inst)"
            >
              <StatusDot :status="selectedInstId === instanceUid(inst) ? 'selected' : onlineStatus[instanceUid(inst)] ? 'online' : 'default'" size="xs" />
              <span>{{ inst.name }}</span>
              <span v-if="inst.isRemote" class="text-[9px] text-orange-400 bg-orange-500/10 px-1 rounded ml-0.5">远程</span>
            </button>
          </div>
        </div>

        <!-- Filter Bar -->
        <div class="flex items-center gap-2 px-5 pt-3 pb-2 shrink-0 flex-wrap">
          <Select v-model="logDate" @update:model-value="onDateChange">
            <SelectTrigger class="h-[32px] w-[120px] text-[13px] border-[var(--border)]">
              <Clock class="h-3.5 w-3.5 mr-1" />
              <SelectValue placeholder="今天" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部</SelectItem>
              <SelectItem v-for="d in availableDates" :key="d.value" :value="d.value">{{ d.label }}</SelectItem>
            </SelectContent>
          </Select>
          <Select v-if="activeTab === 'database'" v-model="levelFilter">
            <SelectTrigger class="h-[32px] w-[80px] text-[13px] border-[var(--border)]">
              <SelectValue placeholder="全部" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部</SelectItem>
              <SelectItem value="error">错误</SelectItem>
              <SelectItem value="warning">警告</SelectItem>
              <SelectItem value="info">信息</SelectItem>
            </SelectContent>
          </Select>
          <div class="flex h-[32px] items-center rounded-lg border border-[var(--border)] bg-[var(--surface)] px-2.5 gap-1.5">
            <Search class="h-3.5 w-3.5 text-[var(--text-tertiary)]" />
            <Input v-model="searchKeyword" placeholder="搜索日志..." class="border-0 shadow-none h-[28px] text-[13px] w-[140px] bg-transparent" />
          </div>
          <div class="flex items-center gap-1.5 ml-auto">
            <button class="btn-ghost h-[30px] text-[12px]" @click="activeTab === 'system' ? loadSystemLogs() : loadLogs()">
              <RefreshCw class="h-3.5 w-3.5" />
            </button>
          </div>
        </div>

        <!-- System Log Level Filter -->
        <div v-if="activeTab === 'system'" class="flex items-center gap-1 px-5 pb-2 shrink-0">
          <button
            v-for="t in systemLogLevels" :key="t.value"
            :class="selectedSystemLogLevel === t.value ? 'pill pill-active' : 'pill pill-default'"
            @click="selectedSystemLogLevel = t.value"
          >
            {{ t.label }}
          </button>
        </div>

        <!-- MySQL Log Type Filter -->
        <div v-if="activeTab === 'database' && (selectedInst.type === 'mysql' || selectedInst.type === 'mariadb')" class="flex items-center gap-1 px-5 pb-2 shrink-0">
          <button
            v-for="t in mysqlLogTypes" :key="t.value"
            :class="selectedMysqlLogType === t.value ? 'pill pill-active' : 'pill pill-default'"
            @click="selectedMysqlLogType = t.value; dbLogPage = 1; loadLogs()"
          >
            {{ t.label }}
          </button>
        </div>

        <!-- Log Content -->
        <div class="flex-1 min-h-0 px-5 pb-4">
          <!-- System Logs -->
          <div v-if="activeTab === 'system'" class="h-full flex flex-col">
            <div ref="sysLogScrollRef" class="flex-1 min-h-0 overflow-y-auto log-viewer rounded-xl p-4 font-mono text-[12px] leading-5">
              <!-- Loading -->
              <div v-if="loadingLogs" class="flex items-center justify-center py-16">
                <Loader2 class="h-5 w-5 text-[var(--accent)] animate-spin mr-2" />
                <span class="text-[13px] text-[var(--text-secondary)]">加载中...</span>
              </div>
              <!-- Empty -->
              <div v-else-if="filteredSystemLogs.length === 0" class="empty-state">
                <div class="empty-state-icon"><FileText class="h-8 w-8" /></div>
                <p class="empty-state-text">暂无系统日志</p>
              </div>
              <!-- Log Lines -->
              <div v-else>
                <div v-for="(log, idx) in pagedSystemLogs" :key="idx" class="flex items-start gap-2 py-1 px-2 -mx-2 rounded-md hover:bg-[var(--surface)] transition-colors">
                  <span class="text-[var(--text-tertiary)] shrink-0 text-[12px] font-mono-data">{{ formatLogTime(log.time) }}</span>
                  <span class="shrink-0 min-w-[56px] text-center" :class="systemLogLevelBadgeClass(log.level)">[{{ (log.level || 'INFO').toUpperCase() }}]</span>
                  <span class="text-[var(--accent)] shrink-0 min-w-[80px] text-[12px] font-mono-data">[{{ log.source || 'SYSTEM' }}]</span>
                  <span class="log-viewer-text break-all flex-1 text-[var(--text-primary)]">{{ log.message }}</span>
                </div>
              </div>
            </div>
            <div v-if="filteredSystemLogs.length > 0" class="flex items-center justify-between pt-3 shrink-0">
              <span class="text-[12px] text-[var(--text-tertiary)]">共 {{ filteredSystemLogs.length }} 条</span>
              <div class="flex items-center gap-2">
                <Select v-model="logPageSize" @update:model-value="sysLogPage = 1">
                  <SelectTrigger class="h-[28px] w-[65px] text-[12px]" style="border-color: var(--border); box-shadow: none">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem :value="50">50</SelectItem>
                    <SelectItem :value="100">100</SelectItem>
                    <SelectItem :value="200">200</SelectItem>
                    <SelectItem :value="500">500</SelectItem>
                  </SelectContent>
                </Select>
                <button class="btn-ghost h-[28px] w-[28px] p-0 flex items-center justify-center" :disabled="sysLogPage <= 1" @click="sysLogPage--">
                  <ChevronLeft class="h-3.5 w-3.5" />
                </button>
                <span class="text-[12px] text-[var(--text-tertiary)]">{{ sysLogPage }} / {{ sysLogTotalPages }}</span>
                <button class="btn-ghost h-[28px] w-[28px] p-0 flex items-center justify-center" :disabled="sysLogPage >= sysLogTotalPages" @click="sysLogPage++">
                  <ChevronRight class="h-3.5 w-3.5" />
                </button>
                <div class="w-px h-4 bg-[var(--border)]" />
                <button class="btn-ghost h-[28px] text-[12px] gap-1" @click="exportSystemLogs('txt')">
                  <Download class="h-3.5 w-3.5" />TXT
                </button>
                <button class="btn-ghost h-[28px] text-[12px] gap-1" @click="exportSystemLogs('json')">
                  <Download class="h-3.5 w-3.5" />JSON
                </button>
              </div>
            </div>
          </div>

          <!-- Database Logs -->
          <div v-if="activeTab === 'database'" class="h-full flex flex-col">
            <div ref="dbLogScrollRef" class="flex-1 min-h-0 overflow-y-auto log-viewer rounded-xl p-4 font-mono text-[12px] leading-5">
              <!-- Loading -->
              <div v-if="loadingLogs" class="flex items-center justify-center py-16">
                <Loader2 class="h-5 w-5 text-[var(--accent)] animate-spin mr-2" />
                <span class="text-[13px] text-[var(--text-secondary)]">加载中...</span>
              </div>
              <!-- Empty -->
              <div v-else-if="filteredLogs.length === 0" class="empty-state">
                <div class="empty-state-icon"><FileText class="h-8 w-8" /></div>
                <p class="empty-state-text">暂无日志数据</p>
              </div>
              <!-- Log Lines -->
              <div v-else>
                <div v-for="(log, idx) in pagedLogs" :key="idx" class="flex items-start gap-2 py-1 px-2 -mx-2 rounded-md hover:bg-[var(--surface)] transition-colors">
                  <span class="text-[var(--text-tertiary)] shrink-0 text-[12px] font-mono-data">{{ formatLogTime(log.time) }}</span>
                  <span class="shrink-0 min-w-[56px] text-center" :class="dbLogLevelBadgeClass(log.level)">[{{ (log.level || 'INFO').toUpperCase() }}]</span>
                  <span class="log-viewer-text break-all flex-1 text-[var(--text-primary)]">{{ log.message }}</span>
                </div>
              </div>
            </div>
            <div v-if="filteredLogs.length > 0" class="flex items-center justify-between pt-3 shrink-0">
              <span class="text-[12px] text-[var(--text-tertiary)]">共 {{ filteredLogs.length }} 条</span>
              <div class="flex items-center gap-2">
                <Select v-model="logPageSize" @update:model-value="dbLogPage = 1">
                  <SelectTrigger class="h-[28px] w-[65px] text-[12px]" style="border-color: var(--border); box-shadow: none">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem :value="50">50</SelectItem>
                    <SelectItem :value="100">100</SelectItem>
                    <SelectItem :value="200">200</SelectItem>
                    <SelectItem :value="500">500</SelectItem>
                  </SelectContent>
                </Select>
                <button class="btn-ghost h-[28px] w-[28px] p-0 flex items-center justify-center" :disabled="dbLogPage <= 1" @click="dbLogPage--">
                  <ChevronLeft class="h-3.5 w-3.5" />
                </button>
                <span class="text-[12px] text-[var(--text-tertiary)]">{{ dbLogPage }} / {{ dbLogTotalPages }}</span>
                <button class="btn-ghost h-[28px] w-[28px] p-0 flex items-center justify-center" :disabled="dbLogPage >= dbLogTotalPages" @click="dbLogPage++">
                  <ChevronRight class="h-3.5 w-3.5" />
                </button>
                <div class="w-px h-4 bg-[var(--border)]" />
                <button class="btn-ghost h-[28px] text-[12px] gap-1" @click="exportLogs('txt')">
                  <Download class="h-3.5 w-3.5" />TXT
                </button>
                <button class="btn-ghost h-[28px] text-[12px] gap-1" @click="exportLogs('json')">
                  <Download class="h-3.5 w-3.5" />JSON
                </button>
              </div>
            </div>
          </div>
        </div>
      </template>

      <!-- No Instance Selected -->
      <div v-else class="flex items-center justify-center flex-1">
        <div class="empty-state">
          <div class="empty-state-icon"><FileText class="h-12 w-12" /></div>
          <p class="empty-state-text">请先在左下角选择一个数据库实例</p>
        </div>
      </div>
    </div>

  <Dialog v-model:open="showLogConfigDialog">
    <DialogContent class="sm:max-w-[460px]">
      <DialogTitle class="text-[16px] font-semibold text-[var(--text-primary)]">日志管理</DialogTitle>
      <DialogDescription class="text-[13px] text-[var(--text-secondary)]">配置日志存储路径、保留策略和功能开关</DialogDescription>
      <div class="grid gap-4 py-4">
        <div>
          <label class="text-[13px] font-medium text-[var(--text-primary)] mb-1.5 block">日志存储路径</label>
          <div class="flex gap-2">
            <Input v-model="logStoragePath" placeholder="选择日志存储路径..." class="text-[13px] flex-1 border-[var(--border)]" />
            <button class="btn-secondary h-[32px] px-3 shrink-0" @click="openFolderBrowser">
              <FolderOpen class="h-3.5 w-3.5 mr-1" />浏览
            </button>
          </div>
        </div>
        <div>
          <label class="text-[13px] font-medium text-[var(--text-primary)] mb-1.5 block">日志保留天数</label>
          <Select v-model="logRetentionDays">
            <SelectTrigger class="text-[13px] border-[var(--border)]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem :value="7">7天</SelectItem>
              <SelectItem :value="15">15天</SelectItem>
              <SelectItem :value="30">30天</SelectItem>
              <SelectItem :value="60">60天</SelectItem>
              <SelectItem :value="90">90天</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>
      <div class="flex justify-end gap-2">
        <button class="btn-secondary h-[32px] text-[13px]" @click="showLogConfigDialog = false">取消</button>
        <button class="btn-primary h-[32px] text-[13px]" @click="saveLogConfig">保存</button>
      </div>
    </DialogContent>
  </Dialog>

  <Dialog v-model:open="showClearLogDialog">
    <DialogContent class="sm:max-w-[425px]">
      <div class="flex flex-col gap-y-1.5 text-center sm:text-left">
        <DialogTitle class="text-[15px] text-[var(--text-primary)]">清空系统日志确认</DialogTitle>
        <DialogDescription class="text-[13px] text-[var(--text-secondary)]">确定要清空系统日志吗？此操作不可撤销。</DialogDescription>
      </div>
      <div class="flex justify-end gap-2 mt-4">
        <button class="btn-secondary h-[32px] text-[13px]" @click="showClearLogDialog = false">取消</button>
        <button class="btn-danger h-[32px] text-[13px]" @click="doClearLogs">确定清空</button>
      </div>
    </DialogContent>
  </Dialog>

  <Dialog v-model:open="showFolderBrowser">
    <DialogContent class="sm:max-w-[480px]">
      <DialogTitle class="text-[15px] font-semibold text-[var(--text-primary)]">选择目录</DialogTitle>
      <DialogDescription class="text-[13px] text-[var(--text-secondary)] truncate">{{ browsePath || '/' }}</DialogDescription>
      <div class="border border-[var(--border)] rounded-lg overflow-hidden">
        <div class="max-h-[320px] overflow-y-auto">
          <div
            v-if="browseParent !== '' && browseParent !== browsePath && !browseIsRoot"
            class="flex items-center gap-2 px-3 py-2 cursor-pointer hover:bg-[var(--surface)] border-b border-[var(--border-subtle)]"
            @click="navigateFolder(browseParent)"
          >
            <ArrowUp class="h-3.5 w-3.5 text-[var(--text-tertiary)] shrink-0" />
            <span class="text-[13px] text-[var(--text-tertiary)]">上级目录</span>
          </div>
          <div
            v-for="d in browseDirs"
            :key="d.path"
            class="flex items-center gap-2 px-3 py-2 cursor-pointer hover:bg-[var(--surface)] border-b border-[var(--border-subtle)] last:border-b-0"
            @click="navigateFolder(d.path)"
            @dblclick="selectFolder(d.path)"
          >
            <HardDrive v-if="d.drive" class="h-3.5 w-3.5 text-[var(--accent)] shrink-0" />
            <FolderOpen v-else class="h-3.5 w-3.5 text-amber-500 shrink-0" />
            <span class="text-[13px] text-[var(--text-primary)] flex-1 truncate">{{ d.name }}</span>
            <ChevronRight class="h-3.5 w-3.5 text-[var(--text-tertiary)] shrink-0" />
          </div>
          <div v-if="browseDirs.length === 0" class="flex items-center justify-center py-8 text-[13px] text-[var(--text-tertiary)]">
            此目录没有子目录
          </div>
        </div>
      </div>
      <div class="flex justify-between gap-2 mt-2">
        <div class="flex-1 min-w-0">
          <Input v-model="browsePath" class="text-[13px] h-[32px] border-[var(--border)]" placeholder="输入路径..." />
        </div>
        <button class="btn-secondary h-[32px] text-[13px] shrink-0" @click="navigateFolder(browsePath)">前往</button>
      </div>
      <div class="flex justify-end gap-2 mt-2">
        <button class="btn-secondary h-[32px] text-[13px]" @click="showFolderBrowser = false">取消</button>
        <button class="btn-primary h-[32px] text-[13px]" @click="selectFolder(browsePath)">选择此目录</button>
      </div>
    </DialogContent>
  </Dialog>
  </div>
</template>

<script setup>
defineOptions({ name: 'LogsView' })
import { ref, computed, onMounted, onActivated, watch, inject, nextTick } from 'vue'
import { storeToRefs } from 'pinia'
import { useAppContext } from '../stores/context'
import { useHealthStore } from '../stores/health'
import { writeLog } from '../api/log'
import { sourceParam, instanceUid } from '@/lib/instance'
import { formatLogTime, isSqlType } from '@/lib/utils'
import { useMessage } from '../composables/useMessage'
import {
  FileText, Clock, Database, Server, Search, Settings,
  RefreshCw, Download, Loader2, Trash2, FolderOpen, ChevronRight, ChevronLeft, ArrowUp, HardDrive
} from 'lucide-vue-next'

import { Button } from '@/components/ui/Button.vue'
import { Input } from '@/components/ui/Input.vue'
import { Dialog, DialogContent, DialogTitle, DialogDescription } from '@/components/ui/Dialog.vue'

const completeProgress = inject('completeProgress')
import StatusDot from './StatusDot.vue'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/Select.vue'
import { STORAGE_KEYS, safeStorage } from '@/lib/storageKeys'

const props = defineProps({
  navRequest: { type: Object, default: null }
})
const emit = defineEmits(['navAccepted'])

const store = useAppContext()
const { connectionId, logEnabled } = storeToRefs(store)

const { success, error, warning } = useMessage()

const activeTab = ref('system')
const selectedInstId = ref(null)
const logDate = ref('today')
const todayKey = ref(new Date().toISOString().slice(0, 10))
const levelFilter = ref('all')
const searchKeyword = ref('')
const selectedMysqlLogType = ref('general')
const loadingLogs = ref(false)
const rawLogs = ref([])
const systemLogs = ref([])
const allInstances = ref([])
const showLogConfigDialog = ref(false)
const logStoragePath = ref('')
const logRetentionDays = ref(30)
const defaultLogPath = ref('./data/logs')
const healthStore = useHealthStore()
const { statusMap: onlineStatus } = storeToRefs(healthStore)
const showClearLogDialog = ref(false)
const showFolderBrowser = ref(false)
const browsePath = ref('')
const browseParent = ref('')
const browseDirs = ref([])
const browseIsRoot = ref(false)

// 分页相关
const sysLogPage = ref(1)
const dbLogPage = ref(1)
const logPageSize = ref(100)
const sysLogScrollRef = ref(null)
const dbLogScrollRef = ref(null)

const mysqlLogTypes = [
  { value: 'general', label: '通用日志' },
  { value: 'error', label: '错误日志' },
  { value: 'slow', label: '慢查询' },
]

const systemLogLevels = [
  { value: 'all', label: '全部' },
  { value: 'error', label: '错误' },
  { value: 'warning', label: '警告' },
  { value: 'info', label: '信息' },
]

const selectedSystemLogLevel = ref('all')

let loadRequestId = 0
let sysLoadRequestId = 0

const availableDates = computed(() => {
  void todayKey.value // 响应式依赖，确保日期变化时重新计算
  const dates = []
  const today = new Date()
  for (let i = 0; i < 7; i++) {
    const d = new Date(today)
    d.setDate(d.getDate() - i)
    const val = i === 0 ? 'today' : `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
    const label = i === 0 ? '今天' : i === 1 ? '昨天' : `${d.getMonth() + 1}月${d.getDate()}日`
    dates.push({ value: val, label })
  }
  return dates
})

const selectedInst = computed(() => allInstances.value.find(i => instanceUid(i) === selectedInstId.value))

const filteredLogs = computed(() => {
  void todayKey.value
  let logs = [...rawLogs.value]
  if (logDate.value !== 'all' && logDate.value !== 'today') {
    logs = logs.filter(l => l.time?.startsWith(logDate.value))
  } else if (logDate.value === 'today') {
    const todayStr = new Date().toISOString().slice(0, 10)
    logs = logs.filter(l => l.time?.startsWith(todayStr))
  }
  if (levelFilter.value && levelFilter.value !== 'all') {
    logs = logs.filter(l => l.level?.toLowerCase() === levelFilter.value.toLowerCase())
  }
  if (searchKeyword.value.trim()) {
    const kw = searchKeyword.value.toLowerCase()
    logs = logs.filter(l => l.message?.toLowerCase().includes(kw) || l.time?.toLowerCase().includes(kw))
  }
  return logs
})

const filteredSystemLogs = computed(() => {
  void todayKey.value
  let logs = [...systemLogs.value]
  if (logDate.value !== 'all' && logDate.value !== 'today') {
    logs = logs.filter(l => l.time?.startsWith(logDate.value))
  } else if (logDate.value === 'today') {
    const todayStr = new Date().toISOString().slice(0, 10)
    logs = logs.filter(l => l.time?.startsWith(todayStr))
  }
  if (selectedSystemLogLevel.value && selectedSystemLogLevel.value !== 'all') {
    logs = logs.filter(l => l.level?.toLowerCase() === selectedSystemLogLevel.value.toLowerCase())
  }
  if (searchKeyword.value.trim()) {
    const kw = searchKeyword.value.toLowerCase()
    logs = logs.filter(l => l.message?.toLowerCase().includes(kw) || l.time?.toLowerCase().includes(kw))
  }
  return logs
})

// 分页后的系统日志
const pagedSystemLogs = computed(() => {
  const start = (sysLogPage.value - 1) * logPageSize.value
  return filteredSystemLogs.value.slice(start, start + logPageSize.value)
})
const sysLogTotalPages = computed(() => Math.max(1, Math.ceil(filteredSystemLogs.value.length / logPageSize.value)))

// 分页后的数据库日志
const pagedLogs = computed(() => {
  const start = (dbLogPage.value - 1) * logPageSize.value
  return filteredLogs.value.slice(start, start + logPageSize.value)
})
const dbLogTotalPages = computed(() => Math.max(1, Math.ceil(filteredLogs.value.length / logPageSize.value)))

const loadInstances = () => {
  const p1 = fetch('/api/databases/db/list/all').then(res => res.json()).then(d => d.code === 0 ? d.data || [] : []).catch(() => [])
  const p2 = fetch('/api/remote-servers').then(res => res.json()).then(d => d.code === 0 ? (d.data || []).map(s => ({ ...s, isRemote: true })) : []).catch(() => [])
  Promise.all([p1, p2]).then(([local, remote]) => {
    allInstances.value = [...local, ...remote]
    checkOnlineStatus(allInstances.value)
    if (store.connectionId && allInstances.value.find(d => instanceUid(d) === store.connectionId)) {
      selectedInstId.value = store.connectionId
      loadSystemLogs()
      loadLogs()
    } else if (allInstances.value.length > 0) {
      selectedInstId.value = instanceUid(allInstances.value[0])
      loadSystemLogs()
      loadLogs()
    }
    completeProgress?.()
  })
}

const checkOnlineStatus = (items) => {
  // 非阻塞刷新
  healthStore.refreshAll()
}

const selectInstance = (inst) => {
  selectedInstId.value = instanceUid(inst)
  store.setContext({
    connectionId: instanceUid(inst),
    userName: inst.username || '',
    dbName: '',
    type: inst.type || 'mysql',
    host: inst.host || '',
    port: inst.port || 0,
    isRemote: inst.isRemote || false,
    name: inst.name || '',
  })
  loadSystemLogs()
  loadLogs()
}

const loadLogs = () => {
  if (!selectedInstId.value) return
  const requestId = ++loadRequestId
  loadingLogs.value = true
  rawLogs.value = []
  const inst = allInstances.value.find(i => instanceUid(i) === selectedInstId.value)
  if (!inst) { loadingLogs.value = false; return }

  if (inst.type === 'mysql' || inst.type === 'mariadb') {
    fetch(`/api/mysql/logs?server_id=${inst.id}&${sourceParam(inst.isRemote || false)}&type=${selectedMysqlLogType.value}`)
      .then(res => res.json()).then(data => {
        if (requestId !== loadRequestId) return
        if (data.code === 0) { rawLogs.value = data.data || [] } else { error(data.msg || '加载失败') }
        loadingLogs.value = false
      }).catch(() => { if (requestId !== loadRequestId) return; error('加载失败'); rawLogs.value = []; loadingLogs.value = false })
  } else if (inst.type === 'postgresql') {
    fetch(`/api/postgresql/logs?server_id=${inst.id}&${sourceParam(inst.isRemote || false)}`)
      .then(res => res.json()).then(data => {
        if (requestId !== loadRequestId) return
        if (data.code === 0) { rawLogs.value = data.data || [] } else { error(data.msg || '加载失败') }
        loadingLogs.value = false
      }).catch(() => { if (requestId !== loadRequestId) return; error('加载失败'); rawLogs.value = []; loadingLogs.value = false })
  } else if (inst.type === 'redis') {
    fetch(`/api/redis/logs?server_id=${inst.id}&${sourceParam(inst.isRemote || false)}`)
      .then(res => res.json()).then(data => {
        if (requestId !== loadRequestId) return
        if (data.code === 0) { rawLogs.value = data.data || [] } else { error(data.msg || '加载失败') }
        loadingLogs.value = false
      }).catch(() => { if (requestId !== loadRequestId) return; error('加载失败'); rawLogs.value = []; loadingLogs.value = false })
  } else if (inst.type === 'sqlite') {
    rawLogs.value = [{ time: '', level: 'Note', message: 'SQLite 是嵌入式文件数据库，没有服务端日志' }]
    loadingLogs.value = false
  } else {
    rawLogs.value = []
    loadingLogs.value = false
  }
}

const onDateChange = () => { loadLogs(); loadSystemLogs(); sysLogPage.value = 1; dbLogPage.value = 1 }

const logLevelClass = (level) => {
  const map = { Note: 'text-emerald-400', Warning: 'text-amber-400', System: 'text-blue-400', ERROR: 'text-red-400', error: 'text-red-400', warning: 'text-amber-400', info: 'text-blue-400' }
  return map[level] || 'text-gray-400'
}

const systemLogLevelClass = (level) => {
  const map = { info: 'text-blue-400', warning: 'text-amber-400', error: 'text-red-400', debug: 'text-muted-foreground' }
  return map[(level || '').toLowerCase()] || 'text-blue-400'
}

// Badge classes using the design system
const systemLogLevelBadgeClass = (level) => {
  const l = (level || '').toLowerCase()
  if (l === 'error') return 'badge-status badge-status-error'
  if (l === 'warning') return 'badge-status badge-status-warning'
  return 'badge-status badge-status-info'
}

const dbLogLevelBadgeClass = (level) => {
  const l = (level || '').toLowerCase()
  if (l === 'error' || level === 'ERROR') return 'badge-status badge-status-error'
  if (l === 'warning' || level === 'Warning') return 'badge-status badge-status-warning'
  return 'badge-status badge-status-info'
}

const loadSystemLogs = () => {
  const requestId = ++sysLoadRequestId
  loadingLogs.value = true
  systemLogs.value = []
  let url = '/api/system/logs'
  const inst = allInstances.value.find(i => instanceUid(i) === selectedInstId.value)
  if (inst) {
    url += `?server_id=${inst.id}&${sourceParam(inst.isRemote || false)}`
  }
  fetch(url)
    .then(res => res.json())
    .then(data => {
      if (requestId !== sysLoadRequestId) return
      if (data.code === 0) {
        systemLogs.value = (data.data || []).map(l => ({
          time: l.time || l.created_at || '',
          level: l.level || 'info',
          source: l.source || 'SYSTEM',
          message: l.message || l.msg || ''
        }))
      } else {
        systemLogs.value = []
      }
      loadingLogs.value = false
    })
    .catch(() => { if (requestId !== sysLoadRequestId) return; systemLogs.value = []; loadingLogs.value = false })
}

const exportSystemLogs = (format) => {
  if (filteredSystemLogs.value.length === 0) { warning('没有可导出的日志'); return }
  let content, filename, mimeType
  if (format === 'json') {
    content = JSON.stringify(filteredSystemLogs.value, null, 2)
    filename = `system_logs_${logDate.value}_${Date.now()}.json`
    mimeType = 'application/json'
  } else {
    content = filteredSystemLogs.value.map(l => `${l.time} [${(l.level || 'INFO').toUpperCase()}] [${l.source || 'SYSTEM'}] ${l.message}`).join('\n')
    filename = `system_logs_${logDate.value}_${Date.now()}.txt`
    mimeType = 'text/plain'
  }
  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = filename; document.body.appendChild(a); a.click(); document.body.removeChild(a)
  URL.revokeObjectURL(url)
  success('导出成功')
}

const confirmClearSystemLogs = () => {
  showClearLogDialog.value = true
}

const doClearLogs = () => {
  fetch('/api/system/logs/clear', { method: 'POST' })
    .then(res => res.json()).then(data => {
      if (data.code === 0) { systemLogs.value = []; success('系统日志已清空'); writeLog('清空系统日志', 'warning') } else { error(data.msg || '清空失败') }
    }).catch(() => error('清空失败'))
    .finally(() => { showClearLogDialog.value = false })
}

const exportLogs = (format) => {
  if (filteredLogs.value.length === 0) { warning('没有可导出的日志'); return }
  let content, filename, mimeType
  if (format === 'json') {
    content = JSON.stringify(filteredLogs.value, null, 2)
    filename = `db_logs_${logDate.value}_${Date.now()}.json`
    mimeType = 'application/json'
  } else {
    content = filteredLogs.value.map(l => `${l.time} [${(l.level || 'INFO').toUpperCase()}] ${l.message}`).join('\n')
    filename = `db_logs_${logDate.value}_${Date.now()}.txt`
    mimeType = 'text/plain'
  }
  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = filename; document.body.appendChild(a); a.click(); document.body.removeChild(a)
  URL.revokeObjectURL(url)
  success('导出成功')
}

const loadLogConfig = () => {
  try {
    const raw = safeStorage.get(STORAGE_KEYS.LOG_CONFIG)
    if (raw) {
      const cfg = JSON.parse(raw)
      logStoragePath.value = cfg.path || ''
      logRetentionDays.value = cfg.retentionDays || 30
    }
  } catch (e) { console.error(e) }
}

const loadLogConfigFromBackend = async () => {
  try {
    const res = await fetch('/api/log-config')
    const data = await res.json()
    if (data.code === 0 && data.data) {
      if (data.data.path) {
        logStoragePath.value = data.data.path
        defaultLogPath.value = data.data.path
      }
      if (data.data.retentionDays) logRetentionDays.value = data.data.retentionDays
    }
  } catch (e) { console.error(e) }
}

const saveLogConfig = () => {
  const cfg = {
    path: logStoragePath.value,
    retentionDays: logRetentionDays.value,
    enabled: logEnabled.value,
  }
  try {
    safeStorage.set(STORAGE_KEYS.LOG_CONFIG, JSON.stringify(cfg))
  } catch (e) { console.error(e) }
  fetch('/api/log-config', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(cfg)
  }).catch(e => console.error(e))
  success('日志配置已保存')
  showLogConfigDialog.value = false
}

const openFolderBrowser = () => {
  browsePath.value = logStoragePath.value || './data/logs'
  loadBrowseDirs(browsePath.value)
  showFolderBrowser.value = true
}

const loadBrowseDirs = (path) => {
  fetch(`/api/fs/browse?path=${encodeURIComponent(path)}`)
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        browsePath.value = data.path
        browseParent.value = data.parent || ''
        browseDirs.value = data.dirs || []
        browseIsRoot.value = data.isRoot === true
      } else {
        error(data.msg || '无法读取目录')
      }
    })
    .catch(() => error('目录浏览请求失败'))
}

const navigateFolder = (path) => {
  loadBrowseDirs(path)
}

const selectFolder = (path) => {
  logStoragePath.value = path
  showFolderBrowser.value = false
}

const processNavRequest = () => {
  if (!props.navRequest) return
  const inst = allInstances.value.find(i => instanceUid(i) === instanceUid(props.navRequest))
  if (inst) {
    selectInstance(inst)
  }
  emit('navAccepted')
}

watch(connectionId, (uid) => {
  if (uid && uid !== selectedInstId.value) {
    selectedInstId.value = uid
    loadSystemLogs()
    loadLogs()
  }
})

watch(activeTab, () => { sysLogPage.value = 1; dbLogPage.value = 1 })
watch(searchKeyword, () => { sysLogPage.value = 1; dbLogPage.value = 1 })
watch(selectedSystemLogLevel, () => { sysLogPage.value = 1 })
watch(levelFilter, () => { dbLogPage.value = 1 })

watch(() => props.navRequest, (val) => { if (val) processNavRequest() })

onMounted(() => { loadInstances(); loadLogConfig(); loadLogConfigFromBackend() })
onActivated(() => { todayKey.value = new Date().toISOString().slice(0, 10); logDate.value = 'today'; if (allInstances.value.length === 0) loadInstances(); loadLogConfig() })
</script>
