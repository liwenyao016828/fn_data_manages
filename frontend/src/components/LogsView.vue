<template>
  <div class="page-padding h-full flex flex-col overflow-hidden">
    <div class="content-card flex flex-col flex-1 min-h-0">
      <div class="content-header shrink-0">
        <div class="flex items-start justify-between">
          <div>
            <h2 class="text-[15px] font-semibold text-foreground">日志中心</h2>
            <p class="text-[13px] text-muted-foreground mt-0.5">查看系统运行日志与数据库日志</p>
          </div>
        </div>
      </div>

      <template v-if="selectedInst">
        <div class="border-t border-border section-padding shrink-0">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <button
                :class="[
                  'inline-flex items-center gap-2 px-5 py-2.5 rounded-xl text-[13px] font-semibold transition-all duration-200 relative overflow-hidden',
                  activeTab === 'system'
                    ? 'tab-active shadow-lg shadow-blue-500/20'
                    : 'tab-inactive'
                ]"
                @click="activeTab = 'system'"
              >
                <FileText class="h-4 w-4" />系统日志
                <span v-if="activeTab === 'system'" class="absolute inset-0 bg-white/10 animate-pulse pointer-events-none"></span>
              </button>
              <button
                v-if="selectedInst.type === 'mysql' || selectedInst.type === 'redis'"
                :class="[
                  'inline-flex items-center gap-2 px-5 py-2.5 rounded-xl text-[13px] font-semibold transition-all duration-200 relative overflow-hidden',
                  activeTab === 'database'
                    ? 'tab-active shadow-lg shadow-blue-500/20'
                    : 'tab-inactive'
                ]"
                @click="activeTab = 'database'"
              >
                <Database v-if="selectedInst.type === 'mysql'" class="h-4 w-4" />
                <Server v-else class="h-4 w-4" />数据库日志
                <span v-if="activeTab === 'database'" class="absolute inset-0 bg-white/10 animate-pulse pointer-events-none"></span>
              </button>
            </div>
            <div class="flex items-center gap-2 flex-wrap">
              <div class="flex h-[32px] items-center rounded-lg border border-border bg-card px-2 gap-1">
                <Search class="h-3.5 w-3.5 text-muted-foreground" />
                <Input v-model="searchKeyword" placeholder="搜索日志..." class="border-0 shadow-none h-[28px] text-[13px] w-[140px] bg-transparent" />
              </div>
              <Select v-model="logDate" @update:model-value="onDateChange">
                <SelectTrigger class="h-[32px] w-[120px] text-[13px] border-border">
                  <Clock class="h-3.5 w-3.5 mr-1" />
                  <SelectValue placeholder="今天" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部</SelectItem>
                  <SelectItem v-for="d in availableDates" :key="d.value" :value="d.value">{{ d.label }}</SelectItem>
                </SelectContent>
              </Select>
              <Select v-if="activeTab === 'database'" v-model="levelFilter">
                <SelectTrigger class="h-[32px] w-[80px] text-[13px] border-border">
                  <SelectValue placeholder="全部" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部</SelectItem>
                  <SelectItem value="error">错误</SelectItem>
                  <SelectItem value="warning">警告</SelectItem>
                  <SelectItem value="info">信息</SelectItem>
                </SelectContent>
              </Select>
              <Button v-if="activeTab === 'system'" variant="destructive" size="sm" class="h-[32px] text-[13px]" @click="confirmClearSystemLogs">
                <Trash2 class="h-3.5 w-3.5 mr-1.5" />清空
              </Button>
              <Button variant="outline" size="sm" class="h-[32px] text-[13px]" @click="activeTab === 'system' ? loadSystemLogs() : loadLogs()">
                <RefreshCw class="h-3.5 w-3.5 mr-1.5" />刷新
              </Button>
            </div>
          </div>
        </div>

        <div v-if="activeTab === 'database'" class="border-t border-border section-padding shrink-0">
          <div class="flex items-center gap-3">
            <div class="flex gap-1.5 flex-1 overflow-x-auto">
              <div
                v-for="inst in allInstances"
                :key="(inst.isRemote ? 'r:' : 'l:') + inst.id"
                :class="[
                  'flex items-center gap-1.5 px-3 py-1.5 rounded-full border cursor-pointer transition-all duration-200 text-xs whitespace-nowrap',
                  selectedInstId === instanceUid(inst)
                    ? 'border-primary/40 bg-primary/8 shadow-sm shadow-primary/10'
                    : 'border-border bg-card hover:border-primary/30 hover:shadow-sm',
                ]"
                @click="selectInstance(inst)"
              >
                <StatusDot :status="selectedInstId === instanceUid(inst) ? 'selected' : onlineStatus[instanceUid(inst)] ? 'online' : 'default'" size="xs" />
                <span class="font-medium text-foreground">{{ inst.name }}</span>
                <span v-if="inst.isRemote" class="text-[9px] text-orange-400 bg-orange-500/5 px-1 rounded">远程</span>
              </div>
            </div>
          </div>
        </div>

        <div v-if="activeTab === 'system'" class="border-t border-border section-padding shrink-0"></div>
        <div class="flex-1 min-h-0" style="padding: 0 var(--section-padding-x) var(--section-padding-y)">
          <div v-if="activeTab === 'system'" class="h-full flex flex-col">
            <div class="mb-2 flex items-center gap-2 shrink-0">
              <span class="text-[12px] text-muted-foreground">日志级别：</span>
              <Button v-for="t in systemLogLevels" :key="t.value" variant="ghost" size="sm"
                :class="[selectedSystemLogLevel === t.value ? 'tab-active' : 'text-muted-foreground hover:bg-muted', 'h-[28px] text-[12px]']"
                @click="selectedSystemLogLevel = t.value">
                {{ t.label }}
              </Button>
            </div>
            <div class="flex-1 min-h-0 overflow-y-auto log-viewer rounded-xl p-3 font-mono text-[12px] leading-5">
              <div v-if="loadingLogs" class="flex items-center justify-center py-16">
                <Loader2 class="h-5 w-5 text-primary animate-spin mr-2" />
                <span class="text-[13px] text-primary">加载中...</span>
              </div>
              <div v-else-if="filteredSystemLogs.length === 0" class="flex items-center justify-center py-16">
                <FileText class="h-8 w-8 text-muted-foreground/40 mr-2" />
                <span class="text-[13px] text-muted-foreground">暂无系统日志</span>
              </div>
              <div v-else>
                <div v-for="(log, idx) in filteredSystemLogs" :key="idx" class="flex items-start gap-2 py-0.5">
                  <span class="text-gray-500/60 shrink-0 text-[12px]">{{ formatLogTime(log.time) }}</span>
                  <span class="shrink-0 min-w-[56px] text-center" :class="systemLogLevelClass(log.level)">[{{ (log.level || 'INFO').toUpperCase() }}]</span>
                  <span class="text-primary shrink-0 min-w-[80px] text-[12px]">[{{ log.source || 'SYSTEM' }}]</span>
                  <span class="log-viewer-text break-all flex-1">{{ log.message }}</span>
                </div>
              </div>
            </div>
            <div v-if="filteredSystemLogs.length > 0" class="flex items-center justify-between mt-2 shrink-0">
              <span class="text-[12px] text-muted-foreground">共 {{ filteredSystemLogs.length }} 条</span>
              <div class="flex items-center gap-2">
                <Button variant="outline" size="sm" class="h-[28px] text-[12px]" @click="exportSystemLogs('txt')">
                  <Download class="h-3.5 w-3.5 mr-1" />导出TXT
                </Button>
                <Button variant="outline" size="sm" class="h-[28px] text-[12px]" @click="exportSystemLogs('json')">
                  <Download class="h-3.5 w-3.5 mr-1" />导出JSON
                </Button>
              </div>
            </div>
          </div>

          <div v-if="activeTab === 'database'" class="h-full flex flex-col">
            <div v-if="selectedInst.type === 'mysql'" class="mb-2 flex items-center gap-2 shrink-0">
              <span class="text-[12px] text-muted-foreground">日志类型：</span>
              <Button v-for="t in mysqlLogTypes" :key="t.value" variant="ghost" size="sm"
                :class="[selectedMysqlLogType === t.value ? 'tab-active' : 'text-muted-foreground hover:bg-muted', 'h-[28px] text-[12px]']"
                @click="selectedMysqlLogType = t.value; loadLogs()">
                {{ t.label }}
              </Button>
            </div>
            <div class="flex-1 min-h-0 overflow-y-auto log-viewer rounded-xl p-3 font-mono text-[12px] leading-5">
              <div v-if="loadingLogs" class="flex items-center justify-center py-16">
                <Loader2 class="h-5 w-5 text-emerald-400 animate-spin mr-2" />
                <span class="text-[13px] text-emerald-400">加载中...</span>
              </div>
              <div v-else-if="filteredLogs.length === 0" class="flex items-center justify-center py-16">
                <FileText class="h-8 w-8 text-muted-foreground/40 mr-2" />
                <span class="text-[13px] text-muted-foreground">暂无日志数据</span>
              </div>
              <div v-else>
                <div v-for="(log, idx) in filteredLogs" :key="idx" class="flex items-start gap-2 py-0.5">
                  <span class="text-gray-500/60 shrink-0 text-[12px]">{{ formatLogTime(log.time) }}</span>
                  <span class="shrink-0 min-w-[56px] text-center" :class="logLevelClass(log.level)">[{{ (log.level || 'INFO').toUpperCase() }}]</span>
                  <span class="log-viewer-text break-all flex-1">{{ log.message }}</span>
                </div>
              </div>
            </div>
            <div v-if="filteredLogs.length > 0" class="flex items-center justify-between mt-2 shrink-0">
              <span class="text-[12px] text-muted-foreground">共 {{ filteredLogs.length }} 条</span>
              <div class="flex items-center gap-2">
                <Button variant="outline" size="sm" class="h-[28px] text-[12px]" @click="exportLogs('txt')">
                  <Download class="h-3.5 w-3.5 mr-1" />导出TXT
                </Button>
                <Button variant="outline" size="sm" class="h-[28px] text-[12px]" @click="exportLogs('json')">
                  <Download class="h-3.5 w-3.5 mr-1" />导出JSON
                </Button>
              </div>
            </div>
          </div>
        </div>
      </template>

      <div v-else class="flex items-center justify-center flex-1">
        <div class="flex flex-col items-center text-muted-foreground">
          <FileText class="h-16 w-16 mb-3 opacity-30" />
          <p class="text-sm">请先在左下角选择一个数据库实例</p>
        </div>
      </div>
    </div>

  <Dialog v-model:open="showLogConfigDialog">
    <DialogContent class="sm:max-w-[460px]">
      <DialogTitle class="text-[16px] font-semibold">日志管理</DialogTitle>
      <DialogDescription class="text-[13px] text-muted-foreground">配置日志存储路径、保留策略和功能开关</DialogDescription>
      <div class="grid gap-4 py-4">
        <div>
          <label class="text-[13px] font-medium text-foreground mb-1.5 block">日志存储路径</label>
          <div class="flex gap-2">
            <Input v-model="logStoragePath" placeholder="选择日志存储路径..." class="text-[13px] flex-1" />
            <Button variant="outline" class="h-[32px] px-3 shrink-0" @click="openFolderBrowser">
              <FolderOpen class="h-3.5 w-3.5 mr-1" />浏览
            </Button>
          </div>
        </div>
        <div>
          <label class="text-[13px] font-medium text-foreground mb-1.5 block">日志保留天数</label>
          <Select v-model="logRetentionDays">
            <SelectTrigger class="text-[13px]">
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
        <Button variant="outline" class="h-[32px] text-[13px]" @click="showLogConfigDialog = false">取消</Button>
        <Button variant="primary" class="h-[32px] text-[13px]" @click="saveLogConfig">保存</Button>
      </div>
    </DialogContent>
  </Dialog>

  <Dialog v-model:open="showClearLogDialog">
    <DialogContent class="sm:max-w-[425px]">
      <div class="flex flex-col gap-y-1.5 text-center sm:text-left">
        <DialogTitle class="text-[15px] text-foreground">清空系统日志确认</DialogTitle>
        <DialogDescription class="text-[13px] text-muted-foreground">确定要清空系统日志吗？此操作不可撤销。</DialogDescription>
      </div>
      <div class="flex justify-end gap-2 mt-4">
        <Button variant="outline" class="h-[32px] text-[13px]" @click="showClearLogDialog = false">取消</Button>
        <Button variant="destructive" class="h-[32px] text-[13px]" @click="doClearLogs">确定清空</Button>
      </div>
    </DialogContent>
  </Dialog>

  <Dialog v-model:open="showFolderBrowser">
    <DialogContent class="sm:max-w-[480px]">
      <DialogTitle class="text-[15px] font-semibold">选择目录</DialogTitle>
      <DialogDescription class="text-[13px] text-muted-foreground truncate">{{ browsePath || '/' }}</DialogDescription>
      <div class="border border-border rounded-lg overflow-hidden">
        <div class="max-h-[320px] overflow-y-auto">
          <div
            v-if="browseParent !== '' && browseParent !== browsePath && !browseIsRoot"
            class="flex items-center gap-2 px-3 py-2 cursor-pointer hover:bg-muted border-b border-border"
            @click="navigateFolder(browseParent)"
          >
            <ArrowUp class="h-3.5 w-3.5 text-muted-foreground shrink-0" />
            <span class="text-[13px] text-muted-foreground">上级目录</span>
          </div>
          <div
            v-for="d in browseDirs"
            :key="d.path"
            class="flex items-center gap-2 px-3 py-2 cursor-pointer hover:bg-muted border-b border-border last:border-b-0"
            @click="navigateFolder(d.path)"
            @dblclick="selectFolder(d.path)"
          >
            <HardDrive v-if="d.drive" class="h-3.5 w-3.5 text-primary shrink-0" />
            <FolderOpen v-else class="h-3.5 w-3.5 icon-special-color shrink-0" />
            <span class="text-[13px] text-foreground flex-1 truncate">{{ d.name }}</span>
            <ChevronRight class="h-3.5 w-3.5 text-muted-foreground shrink-0" />
          </div>
          <div v-if="browseDirs.length === 0" class="flex items-center justify-center py-8 text-[13px] text-muted-foreground">
            此目录没有子目录
          </div>
        </div>
      </div>
      <div class="flex justify-between gap-2 mt-2">
        <div class="flex-1 min-w-0">
          <Input v-model="browsePath" class="text-[13px] h-[32px]" placeholder="输入路径..." />
        </div>
        <Button variant="outline" class="h-[32px] text-[13px] shrink-0" @click="navigateFolder(browsePath)">前往</Button>
      </div>
      <div class="flex justify-end gap-2 mt-2">
        <Button variant="outline" class="h-[32px] text-[13px]" @click="showFolderBrowser = false">取消</Button>
        <Button variant="primary" class="h-[32px] text-[13px]" @click="selectFolder(browsePath)">选择此目录</Button>
      </div>
    </DialogContent>
  </Dialog>
  </div>
</template>

<script setup>
defineOptions({ name: 'LogsView' })
import { ref, computed, onMounted, onActivated, watch, inject } from 'vue'
import { storeToRefs } from 'pinia'
import { useAppContext } from '../stores/context'
import { useHealthStore } from '../stores/health'
import { writeLog } from '../api/log'
import { sourceParam, instanceUid } from '@/lib/instance'
import { formatLogTime } from '@/lib/utils'
import { useMessage } from '../composables/useMessage'
import {
  FileText, Clock, Database, Server, Search, Settings,
  RefreshCw, Download, Loader2, Trash2, FolderOpen, ChevronRight, ArrowUp, HardDrive
} from 'lucide-vue-next'

import { Button } from '@/components/ui/Button.vue'
import { Input } from '@/components/ui/Input.vue'
import { Dialog, DialogContent, DialogTitle, DialogDescription } from '@/components/ui/Dialog.vue'

const completeProgress = inject('completeProgress')
import StatusDot from './StatusDot.vue'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/Select.vue'

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
const levelFilter = ref('all')
const searchKeyword = ref('')
const selectedMysqlLogType = ref('error')
const loadingLogs = ref(false)
const rawLogs = ref([])
const systemLogs = ref([])
const allInstances = ref([])
const showLogConfigDialog = ref(false)
const logStoragePath = ref('./data/logs')
const logRetentionDays = ref(30)
const healthStore = useHealthStore()
const { statusMap: onlineStatus } = storeToRefs(healthStore)
const showClearLogDialog = ref(false)
const showFolderBrowser = ref(false)
const browsePath = ref('')
const browseParent = ref('')
const browseDirs = ref([])
const browseIsRoot = ref(false)

const mysqlLogTypes = [
  { value: 'error', label: '错误日志' },
  { value: 'slow', label: '慢查询' },
  { value: 'general', label: '通用日志' },
]

const systemLogLevels = [
  { value: 'all', label: '全部' },
  { value: 'error', label: '错误' },
  { value: 'warning', label: '警告' },
  { value: 'info', label: '信息' },
]

const selectedSystemLogLevel = ref('all')

const availableDates = computed(() => {
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

const checkOnlineStatus = async (items) => {
  await healthStore.refreshAll()
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
  loadingLogs.value = true
  rawLogs.value = []
  const inst = allInstances.value.find(i => instanceUid(i) === selectedInstId.value)
  if (!inst) { loadingLogs.value = false; return }

  if (inst.type === 'mysql') {
    fetch(`/api/mysql/logs?server_id=${inst.id}&${sourceParam(inst.isRemote || false)}&type=${selectedMysqlLogType.value}`)
      .then(res => res.json()).then(data => {
        if (data.code === 0) { rawLogs.value = data.data || [] } else { error(data.msg || '加载失败') }
        loadingLogs.value = false
      }).catch(() => { error('加载失败'); rawLogs.value = []; loadingLogs.value = false })
  } else {
    fetch(`/api/redis/logs?server_id=${inst.id}&${sourceParam(inst.isRemote || false)}`)
      .then(res => res.json()).then(data => {
        if (data.code === 0) { rawLogs.value = data.data || [] } else { error(data.msg || '加载失败') }
        loadingLogs.value = false
      }).catch(() => { error('加载失败'); rawLogs.value = []; loadingLogs.value = false })
  }
}

const onDateChange = () => { loadLogs(); loadSystemLogs() }

const logLevelClass = (level) => {
  const map = { Note: 'text-emerald-400', Warning: 'text-amber-400', System: 'text-blue-400', ERROR: 'text-red-400', error: 'text-red-400', warning: 'text-amber-400', info: 'text-blue-400' }
  return map[level] || 'text-gray-400'
}

const systemLogLevelClass = (level) => {
  const map = { info: 'text-blue-400', warning: 'text-amber-400', error: 'text-red-400', debug: 'text-muted-foreground' }
  return map[(level || '').toLowerCase()] || 'text-blue-400'
}

const loadSystemLogs = () => {
  let url = '/api/system/logs'
  const inst = allInstances.value.find(i => instanceUid(i) === selectedInstId.value)
  if (inst) {
    url += `?server_id=${inst.id}&${sourceParam(inst.isRemote || false)}`
  }
  fetch(url)
    .then(res => res.json())
    .then(data => {
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
    })
    .catch(() => { systemLogs.value = [] })
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
    const raw = localStorage.getItem('log_config')
    if (raw) {
      const cfg = JSON.parse(raw)
      logStoragePath.value = cfg.path || './data/logs'
      logRetentionDays.value = cfg.retentionDays || 30
    }
  } catch (e) { console.error(e) }
}

const loadLogConfigFromBackend = async () => {
  try {
    const res = await fetch('/api/log-config')
    const data = await res.json()
    if (data.code === 0 && data.data) {
      if (data.data.path) logStoragePath.value = data.data.path
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
    localStorage.setItem('log_config', JSON.stringify(cfg))
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

watch([connectionId], () => {
  loadInstances()
})

watch(() => props.navRequest, (val) => { if (val) processNavRequest() })

onMounted(() => { loadInstances(); loadLogConfig(); loadLogConfigFromBackend() })
onActivated(() => { loadInstances(); loadLogConfig() })
</script>