<template>
  <div class="page-padding h-full overflow-y-auto">
    <!-- Page Header -->
    <div class="flex items-center justify-between section-gap flex-wrap gap-3">
      <div>
        <h2 class="text-lg font-semibold tracking-tight" style="color: var(--text-primary)">连接管理</h2>
        <p class="text-[13px] mt-0.5" style="color: var(--text-tertiary)">管理所有数据库实例连接</p>
      </div>
      <div class="flex items-center gap-2">
        <Button variant="outline" size="sm" class="h-[32px] text-[13px]" @click="showDetectDialog = true">
          <Search class="h-3.5 w-3.5 mr-1.5" />检测
        </Button>
        <Button variant="primary" size="sm" class="h-[32px] text-[13px]" @click="showCreateDialog = true">
          <Plus class="h-3.5 w-3.5 mr-1.5" />添加
        </Button>
      </div>
    </div>

    <!-- Stats Row with Filter -->
    <div class="grid grid-cols-3 md:grid-cols-6 gap-3 section-gap">
      <div
        class="stat-card cursor-pointer transition-all duration-200"
        :class="{ 'ring-2 ring-[var(--text-primary)] bg-[var(--surface-hover)]': activeTab === 'all' }"
        @click="activeTab = 'all'"
      >
        <div class="stat-icon" style="background: var(--surface-hover); color: var(--text-primary)">
          <BarChart3 class="h-4 w-4" />
        </div>
        <div class="flex flex-col">
          <span class="stat-value">{{ stats.total }}</span>
          <span class="stat-label">总计</span>
        </div>
      </div>
      <div
        class="stat-card cursor-pointer transition-all duration-200"
        :class="{ 'ring-2 ring-[#3b82f6] bg-[rgba(59,130,246,0.1)]': activeTab === 'mysql' }"
        @click="activeTab = activeTab === 'mysql' ? 'all' : 'mysql'"
      >
        <div class="stat-icon" style="background: rgba(59, 130, 246, 0.1); color: #3b82f6">
          <Database class="h-4 w-4" />
        </div>
        <div class="flex flex-col">
          <span class="stat-value">{{ stats.mysql }}</span>
          <span class="stat-label">MySQL</span>
        </div>
      </div>
      <div
        class="stat-card cursor-pointer transition-all duration-200"
        :class="{ 'ring-2 ring-[#06b6d4] bg-[rgba(6,182,212,0.1)]': activeTab === 'mariadb' }"
        @click="activeTab = activeTab === 'mariadb' ? 'all' : 'mariadb'"
      >
        <div class="stat-icon" style="background: rgba(6, 182, 212, 0.1); color: #06b6d4">
          <Database class="h-4 w-4" />
        </div>
        <div class="flex flex-col">
          <span class="stat-value">{{ stats.mariadb }}</span>
          <span class="stat-label">MariaDB</span>
        </div>
      </div>
      <div
        class="stat-card cursor-pointer transition-all duration-200"
        :class="{ 'ring-2 ring-[#6366f1] bg-[rgba(99,102,241,0.1)]': activeTab === 'postgresql' }"
        @click="activeTab = activeTab === 'postgresql' ? 'all' : 'postgresql'"
      >
        <div class="stat-icon" style="background: rgba(99, 102, 241, 0.1); color: #6366f1">
          <Database class="h-4 w-4" />
        </div>
        <div class="flex flex-col">
          <span class="stat-value">{{ stats.postgresql }}</span>
          <span class="stat-label">PostgreSQL</span>
        </div>
      </div>
      <div
        class="stat-card cursor-pointer transition-all duration-200"
        :class="{ 'ring-2 ring-[#f59e0b] bg-[rgba(245,158,11,0.1)]': activeTab === 'redis' }"
        @click="activeTab = activeTab === 'redis' ? 'all' : 'redis'"
      >
        <div class="stat-icon" style="background: rgba(245, 158, 11, 0.1); color: #f59e0b">
          <Server class="h-4 w-4" />
        </div>
        <div class="flex flex-col">
          <span class="stat-value">{{ stats.redis }}</span>
          <span class="stat-label">Redis</span>
        </div>
      </div>
      <div
        class="stat-card cursor-pointer transition-all duration-200"
        :class="{ 'ring-2 ring-[#10b981] bg-[rgba(16,185,129,0.1)]': activeTab === 'sqlite' }"
        @click="activeTab = activeTab === 'sqlite' ? 'all' : 'sqlite'"
      >
        <div class="stat-icon" style="background: rgba(16, 185, 129, 0.1); color: #10b981">
          <Database class="h-4 w-4" />
        </div>
        <div class="flex flex-col">
          <span class="stat-value">{{ stats.sqlite }}</span>
          <span class="stat-label">SQLite</span>
        </div>
      </div>
    </div>

    <!-- Search Bar -->
    <div class="flex items-center gap-3 section-gap">
      <div class="flex items-center rounded-xl border px-3 gap-1.5 h-[34px]" style="border-color: var(--border); background: var(--surface)">
        <Search class="h-3.5 w-3.5 shrink-0" style="color: var(--text-tertiary)" />
        <Input v-model="searchKeyword" placeholder="搜索实例..." class="border-0 shadow-none h-[28px] text-[13px] w-[240px] bg-transparent" @input="handleSearch" />
      </div>
      <span v-if="activeTab !== 'all'" class="text-[12px]" style="color: var(--text-tertiary)">
        已筛选: {{ getTypeLabel(activeTab) }}
        <span class="cursor-pointer ml-1" style="color: var(--accent)" @click="activeTab = 'all'">清除</span>
      </span>
    </div>

    <!-- Connection Cards Grid -->
    <div v-if="tableData.length === 0" class="empty-state py-20">
      <Inbox class="h-10 w-10 empty-state-icon" />
      <p class="empty-state-text">暂无数据</p>
      <Button variant="outline" size="sm" class="mt-3 h-[32px] text-[13px]" @click="showCreateDialog = true">
        <Plus class="h-3.5 w-3.5 mr-1.5" />添加连接
      </Button>
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 grid-gap section-gap">
      <div
        v-for="(row, idx) in tableData"
        :key="instanceUid(row)"
        class="conn-card group"
        :class="{ 'conn-card-selected': connectionId === instanceUid(row) }"
        :style="{ '--type-color': getTypeColor(row.type), '--type-soft': getTypeSoftColor(row.type), '--card-stagger': `${idx * 45}ms` }"
        @click="selectConnection(row)"
      >
        <!-- Card Body -->
        <div class="conn-card-body">
          <!-- Header: Name (left) · Type · Source · Status (right) -->
          <div class="conn-head">
            <div class="conn-head-left">
              <span
                v-if="connectionId === instanceUid(row)"
                class="conn-selected-dot"
              />
              <span class="conn-name">{{ row.name }}</span>
            </div>
            <div class="conn-head-right">
              <span class="conn-type">{{ getTypeLabel(row.type) }}</span>
              <span class="conn-dot" />
              <span class="conn-source" :class="row.isRemote ? 'conn-source--remote' : 'conn-source--local'">
                {{ row.isRemote ? '远程' : '本地' }}
              </span>
              <span class="conn-dot" />
              <span class="conn-status" :data-status="connStatusSimple(row)">
                <StatusDot :status="connStatusSimple(row)" size="xs" />
                <span class="conn-status-label">{{ connStatusLabelSimple(row) }}</span>
              </span>
            </div>
          </div>

          <!-- Address line -->
          <div class="conn-address">
            <span class="conn-host">{{ row.host }}:{{ row.port }}</span>
            <span class="conn-sep">·</span>
            <span class="conn-user">{{ row.username || '—' }}</span>
          </div>

          <!-- Flip: Front = Password + Description, Back = Info -->
          <div class="conn-flip" :class="{ 'conn-flip--on': flippedUid === instanceUid(row) }">
            <div class="conn-flip-inner" @click.stop="toggleFlip(row)">
              <!-- Front -->
              <div class="conn-flip-face conn-flip-front">
                <div class="conn-password">
                  <span
                    v-if="passwordState[passwordUid(row)]?.loaded"
                    class="conn-password-val"
                    :title="passwordState[passwordUid(row)]?.value"
                  >{{ passwordState[passwordUid(row)]?.value || '（空）' }}</span>
                  <span v-else class="conn-password-val conn-password-val--masked">••••••</span>
                  <button
                    class="conn-eye"
                    :class="{ 'conn-eye--on': passwordState[passwordUid(row)]?.loaded }"
                    :title="passwordState[passwordUid(row)]?.loaded ? '隐藏密码' : '显示密码'"
                    @click.stop="togglePassword(row)"
                  >
                    <Loader2 v-if="passwordState[passwordUid(row)]?.loading" class="h-3 w-3 animate-spin" />
                    <EyeOff v-else-if="passwordState[passwordUid(row)]?.loaded" class="h-3 w-3" />
                    <Eye v-else class="h-3 w-3" />
                  </button>
                </div>
                <p v-if="row.description" class="conn-desc">{{ row.description }}</p>
                <p v-else class="conn-desc conn-desc--empty">未填写描述</p>
              </div>

              <!-- Back -->
              <div class="conn-flip-face conn-flip-back">
                <div class="conn-detail">
                  <div class="conn-detail-item">
                    <span class="conn-detail-label">权限</span>
                    <span class="conn-detail-val">{{ row.permission || '%' }}</span>
                  </div>
                  <div class="conn-detail-item">
                    <span class="conn-detail-label">SSL</span>
                    <span class="conn-detail-val">{{ row.ssl ? '开启' : '关闭' }}</span>
                  </div>
                  <div class="conn-detail-item">
                    <span class="conn-detail-label">版本</span>
                    <span class="conn-detail-val">{{ row.version || '—' }}</span>
                  </div>
                  <div class="conn-detail-item">
                    <span class="conn-detail-label">磁盘</span>
                    <span class="conn-detail-val">{{ row.disk || '—' }}</span>
                  </div>
                  <div class="conn-detail-item">
                    <span class="conn-detail-label">创建时间</span>
                    <span class="conn-detail-val">{{ formatLogTime(row.createdAt) || '—' }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Duplicate tag -->
          <div v-if="isDuplicate(row)" class="conn-dup">
            <span>重复连接</span>
          </div>

          <!-- Footer actions -->
          <div class="conn-footer">
            <div class="conn-footer-left">
              <button class="conn-btn" @click.stop="navigateTo('data', row)">数据</button>
              <button class="conn-btn" @click.stop="navigateTo('backup', row)">备份</button>
              <button v-if="logEnabled" class="conn-btn" @click.stop="navigateTo('logs', row)">日志</button>
            </div>
            <div class="conn-footer-right">
              <button class="conn-btn conn-btn--edit" @click.stop="handleEdit(row)">
                <Pencil class="h-3 w-3" />编辑
              </button>
              <button class="conn-btn conn-btn--danger" @click.stop="handleDelete(row)">
                <Trash2 class="h-3 w-3" />删除
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Pagination -->
    <div v-if="tableData.length > 0" class="flex items-center justify-between flex-wrap gap-2 pt-4" style="border-top: 1px solid var(--border-subtle)">
      <span class="text-[12px]" style="color: var(--text-tertiary)">共 {{ total }} 条</span>
      <div class="flex items-center gap-2">
        <Select v-model="pageSize" @update:model-value="onPageSizeChange">
          <SelectTrigger class="h-[32px] w-[70px] text-[12px]" style="border-color: var(--border); box-shadow: none">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem :value="10">10</SelectItem>
            <SelectItem :value="20">20</SelectItem>
            <SelectItem :value="50">50</SelectItem>
            <SelectItem :value="100">100</SelectItem>
          </SelectContent>
        </Select>
        <div class="flex items-center gap-1">
          <Button variant="outline" size="icon" class="h-[32px] w-[32px]" :disabled="page <= 1" @click="page--">
            <ChevronLeft class="h-3.5 w-3.5" style="color: var(--text-tertiary)" />
          </Button>
          <span class="text-[12px] px-2" style="color: var(--text-tertiary)">{{ page }} / {{ totalPages }}</span>
          <Button variant="outline" size="icon" class="h-[32px] w-[32px]" :disabled="page >= totalPages" @click="page++">
            <ChevronRight class="h-3.5 w-3.5" style="color: var(--text-tertiary)" />
          </Button>
        </div>
      </div>
    </div>

    <ManagementDialog v-model="showCreateDialog" type="create" :existing-instances="allData" @success="handleSuccess" />
    <ManagementDialog v-model="showEditDialog" type="edit" :data="editData" :existing-instances="allData" @success="handleSuccess" />
    <DetectDialog v-model="showDetectDialog" @success="handleSuccess" />

    <Dialog v-model:open="showDeleteDialog">
      <DialogContent class="sm:max-w-[425px] rounded-xl">
        <div class="flex flex-col gap-y-1.5 text-center sm:text-left">
          <DialogTitle class="text-[15px]">删除确认</DialogTitle>
          <DialogDescription class="text-[13px]" style="color: var(--text-secondary)">
            确定要删除实例 "{{ deleteTarget?.name }}" 吗？此操作不可恢复。
          </DialogDescription>
        </div>
        <div class="flex justify-end gap-2 mt-4">
          <Button variant="outline" class="h-[32px] text-[13px]" @click="showDeleteDialog = false">取消</Button>
          <Button variant="destructive" class="h-[32px] text-[13px]" @click="confirmDelete">确定删除</Button>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
defineOptions({ name: 'ManagementView' })
import { ref, computed, onMounted, onActivated, onUnmounted, watch, inject } from 'vue'
import { storeToRefs } from 'pinia'
import { useAppContext } from '../stores/context'
import { useHealthStore } from '../stores/health'
import { sourceParam, instanceUid } from '@/lib/instance'
import { formatLogTime, getTypeLabel, getTypeColor, getTypeSoftColor } from '@/lib/utils'
import { toast } from 'vue-sonner'
import {
  Search, Plus, Database, Server, BarChart3, Activity, ChevronRight, ChevronLeft, Inbox,
  Eye, EyeOff, Loader2, Pencil, Trash2,
} from 'lucide-vue-next'
import { databaseApi } from '../api/database'
import ManagementDialog from './ManagementDialog.vue'
import DetectDialog from './DetectDialog.vue'
import StatusDot from './StatusDot.vue'
import { Button } from '@/components/ui/Button.vue'

const completeProgress = inject('completeProgress')
import { Badge } from '@/components/ui/Badge.vue'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/Table.vue'
import { Dialog, DialogContent, DialogTitle, DialogDescription } from '@/components/ui/Dialog.vue'
import { Input } from '@/components/ui/Input.vue'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/Tabs.vue'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/Select.vue'

const emit = defineEmits(['navigate'])

const store = useAppContext()
const { connectionId, logEnabled } = storeToRefs(store)

const searchKeyword = ref('')
const activeTab = ref('all')
const allData = ref([])
const page = ref(1)
const pageSize = ref(10)
const showCreateDialog = ref(false)
const showEditDialog = ref(false)
const showDetectDialog = ref(false)
const editData = ref({})
const expandedRowUid = ref(null)
const showDeleteDialog = ref(false)
const deleteTarget = ref(null)
const flippedUid = ref(null)
const stats = ref({ mysql: 0, mariadb: 0, postgresql: 0, redis: 0, sqlite: 0, total: 0, running: 0 })
const healthStore = useHealthStore()
const { statusMap: onlineStatus } = storeToRefs(healthStore)
const previousIds = ref(new Set())
let searchTimer = null
let showTimer = null
let hideTimer = null
const HOVER_DELAY = 600 // 600ms 延迟显示

const filteredData = computed(() => {
  const searchType = activeTab.value === 'all' ? '' : activeTab.value
  const kw = searchKeyword.value.toLowerCase()
  return allData.value.filter(item => {
    if (searchType && item.type !== searchType) return false
    if (kw) {
      const searchable = [
        item.name || '',
        item.host || '',
        item.username || '',
        item.description || ''
      ].join(' ').toLowerCase()
      if (!searchable.includes(kw)) return false
    }
    return true
  })
})

const duplicateInfo = computed(() => {
  const items = allData.value
  // 只使用主机、端口和用户名三者组合来判断重复
  const connectionMap = new Map()
  const duplicateUids = new Set()

  for (const item of items) {
    const uid = instanceUid(item)
    // 构建唯一键：主机:端口:用户名:类型（全部小写）
    const key = `${(item.host || '').toLowerCase()}:${item.port}:${(item.username || '').toLowerCase()}:${(item.type || '').toLowerCase()}`

    if (!connectionMap.has(key)) {
      connectionMap.set(key, [])
    }
    connectionMap.get(key).push({ uid, item })

    if (connectionMap.get(key).length > 1) {
      connectionMap.get(key).forEach(e => duplicateUids.add(e.uid))
    }
  }

  // 收集重复的连接组
  const duplicates = []
  for (const [, entries] of connectionMap) {
    if (entries.length > 1) {
      duplicates.push(entries.map(e => e.item))
    }
  }

  return { duplicateUids, duplicates, count: duplicateUids.size }
})

const isDuplicate = (row) => duplicateInfo.value.duplicateUids.has(instanceUid(row))

const total = computed(() => filteredData.value.length)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const tableData = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredData.value.slice(start, start + pageSize.value)
})

const selectConnection = (row) => {
  store.setContext({
    connectionId: instanceUid(row),
    name: row.name,
    type: row.type,
    host: row.host,
    port: row.port,
    userName: row.username,
    isRemote: row.isRemote || false,
  })
}

const TYPE_ICONS = {
  mysql: Database,
  mariadb: Database,
  postgresql: Database,
  redis: Server,
  sqlite: Database,
}

const getTypeIcon = (type) => TYPE_ICONS[type] || Database

const connStatus = (row) => {
  const uid = instanceUid(row)
  if (connectionId.value === uid) return 'selected'
  const status = onlineStatus.value?.[uid]
  if (status === true) return 'online'
  if (status === false) return 'offline'
  return 'default'
}

const connStatusLabel = (row) => {
  const s = connStatus(row)
  if (s === 'selected') return '已选中'
  if (s === 'online') return '在线'
  if (s === 'offline') return '离线'
  return '未检测'
}

const connStatusSimple = (row) => {
  const status = onlineStatus.value?.[instanceUid(row)]
  if (status === true) return 'online'
  if (status === false) return 'offline'
  return 'default'
}

const connStatusLabelSimple = (row) => {
  const s = connStatusSimple(row)
  if (s === 'online') return '在线'
  if (s === 'offline') return '离线'
  return '未检测'
}

const resetPageAndCollapse = () => {
  page.value = 1
  expandedRowUid.value = null
}

// 密码显示/隐藏：按需请求后端明文
// 状态: 0=未加载, 1=已隐藏, 2=已显示
const passwordState = ref({}) // uid -> { loaded, value, loading }
const passwordUid = (row) => `${row.id}`

const togglePassword = async (row) => {
  const uid = passwordUid(row)
  const cur = passwordState.value[uid]
  if (cur?.loaded && !cur.loading) {
    passwordState.value[uid] = { ...cur, loaded: false, value: '' }
    return
  }
  if (cur?.loading) return
  passwordState.value[uid] = { loaded: false, value: '', loading: true }
  try {
    const res = await databaseApi.revealPassword(row.id)
    passwordState.value[uid] = { loaded: true, value: res.data.data.password || '', loading: false }
  } catch (e) {
    toast.error('获取密码失败')
    passwordState.value[uid] = { loaded: true, value: '', loading: false }
  }
}

const fetchData = () => {
  expandedRowUid.value = null
  const p1 = databaseApi.search({ page: 1, pageSize: 99999, info: '', type: '' })
    .then(res => { if (res.data.code === 0) { return res.data.data.items || [] } return [] }).catch(() => [])

  const p2 = fetch('/api/remote-servers').then(res => res.json()).then(data => {
    if (data.code === 0) return (data.data || []).map(srv => ({ ...srv, isRemote: true, disk: srv.disk || '', version: srv.version || '', type: srv.type || 'mysql' }))
    return []
  }).catch(() => [])

  Promise.all([p1, p2]).then(([local, remote]) => {
    const result = [...local, ...remote]
    const newIds = new Set(result.map(i => instanceUid(i)))
    const newInstanceIds = [...newIds].filter(uid => !previousIds.value.has(uid))
    allData.value = result
    previousIds.value = newIds
    stats.value.total = allData.value.length
    stats.value.mysql = allData.value.filter(i => (i.type || 'mysql') === 'mysql').length
    stats.value.mariadb = allData.value.filter(i => i.type === 'mariadb').length
    stats.value.postgresql = allData.value.filter(i => i.type === 'postgresql').length
    stats.value.redis = allData.value.filter(i => i.type === 'redis').length
    stats.value.sqlite = allData.value.filter(i => i.type === 'sqlite').length
    stats.value.running = Object.values(onlineStatus.value).filter(Boolean).length
    // 仅刷新在线状态，不做逐个 forceCheck（后端已定时检测）
    checkOnlineStatus(allData.value)
    // 延迟补全缺失信息，避免阻塞首屏
    if (autoRefreshMissing(allData.value)) {
      setTimeout(() => checkOnlineStatus(allData.value), 3000)
    }
    completeProgress?.()
  })
}

const checkOnlineStatus = (items) => {
  // 非阻塞刷新
  healthStore.refreshAll()
  stats.value.running = Object.values(onlineStatus.value).filter(Boolean).length
}

const autoRefreshMissing = (items) => {
  const missing = items.filter(item => !item.isRemote && (!item.version || !item.disk))
  if (missing.length === 0) return false
  // 串行逐个补全，每个完成后间隔 500ms，避免同时建立多个数据库连接
  let idx = 0
  const next = () => {
    if (idx >= missing.length) return
    const item = missing[idx++]
    fetch('/api/databases/refresh', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: item.id }) })
      .then(r => r.json()).then(data => {
        if (data.code === 0) {
          const i = allData.value.findIndex(r => r.id === item.id && !r.isRemote)
          if (i >= 0) {
            if (data.data.version && !allData.value[i].version) allData.value[i].version = data.data.version
            if (data.data.disk && !allData.value[i].disk) allData.value[i].disk = data.data.disk
          }
        }
      }).catch(() => {}).finally(() => setTimeout(next, 500))
  }
  next()
  return true
}

const handleSearch = () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    resetPageAndCollapse()
  }, 300)
}
const onPageSizeChange = (val) => { pageSize.value = val; resetPageAndCollapse() }
watch(activeTab, () => { resetPageAndCollapse() })
const handleEdit = (row) => { editData.value = { ...row }; showEditDialog.value = true }
const handleDelete = (row) => { deleteTarget.value = row; showDeleteDialog.value = true }
const handleSuccess = () => { fetchData() }

const navigateTo = (action, row) => {
  emit('navigate', { action, row })
}
const toggleOverlay = (row) => {
  const uid = instanceUid(row)
  clearTimers()
  if (expandedRowUid.value === uid) {
    expandedRowUid.value = null
  } else {
    expandedRowUid.value = uid
  }
}

const toggleFlip = (row) => {
  const uid = instanceUid(row)
  flippedUid.value = flippedUid.value === uid ? null : uid
}
const showOverlayDelay = (row) => {
  clearTimers()
  const uid = instanceUid(row)
  showTimer = setTimeout(() => {
    expandedRowUid.value = uid
  }, HOVER_DELAY)
}
const hideOverlayDelay = (row) => {
  if (showTimer) {
    clearTimeout(showTimer)
    showTimer = null
  }
  hideTimer = setTimeout(() => {
    expandedRowUid.value = null
  }, 150)
}
const clearHideTimer = () => {
  if (hideTimer) {
    clearTimeout(hideTimer)
    hideTimer = null
  }
}
const clearTimers = () => {
  if (searchTimer) {
    clearTimeout(searchTimer)
    searchTimer = null
  }
  if (showTimer) {
    clearTimeout(showTimer)
    showTimer = null
  }
  if (hideTimer) {
    clearTimeout(hideTimer)
    hideTimer = null
  }
}

const confirmDelete = () => {
  if (!deleteTarget.value) return
  if (deleteTarget.value.isRemote) {
    fetch(`/api/remote-servers/${deleteTarget.value.id}`, { method: 'DELETE' }).then(res => res.json()).then(data => {
      if (data.code === 0) { toast.success('删除成功'); fetchData() } else { toast.error(data.msg || '删除失败') }
    }).catch(() => { toast.error('删除失败') }).finally(() => { showDeleteDialog.value = false })
  } else {
    databaseApi.delete({ id: deleteTarget.value.id }).then(res => {
      if (res.data.code === 0) { toast.success('删除成功'); fetchData() } else { toast.error(res.data.msg || '删除失败') }
    }).catch(() => { toast.error('删除失败') }).finally(() => { showDeleteDialog.value = false })
  }
}

onMounted(() => { fetchData() })
onActivated(() => { if (allData.value.length === 0) fetchData() })
onUnmounted(() => { clearTimers() })
</script>

<style scoped>
/* ── Card ──────────────────────────────────────────────────── */
.conn-card {
  position: relative;
  border-radius: 14px;
  background: var(--surface);
  border: 1px solid var(--border-subtle);
  transition:
    transform 260ms cubic-bezier(0.16, 1, 0.3, 1),
    box-shadow 260ms cubic-bezier(0.16, 1, 0.3, 1),
    border-color 200ms ease,
    background 200ms ease;
  cursor: default;
  animation: connCardEnter 440ms cubic-bezier(0.16, 1, 0.3, 1) both;
  animation-delay: var(--card-stagger, 0ms);
  overflow: hidden;
}

.conn-card::before {
  content: '';
  position: absolute;
  left: 0; top: 0; bottom: 0;
  width: 3px;
  background: var(--type-color, var(--accent));
  opacity: 0.85;
  transition: width 260ms cubic-bezier(0.16, 1, 0.3, 1), opacity 200ms ease;
  pointer-events: none;
}

.conn-card:hover {
  transform: translateY(-2px);
  border-color: color-mix(in srgb, var(--type-color, var(--accent)) 28%, var(--border-subtle));
  box-shadow:
    0 1px 2px rgba(15, 23, 42, 0.04),
    0 8px 24px color-mix(in srgb, var(--type-color, var(--accent)) 10%, transparent);
}

.conn-card:hover::before { opacity: 1; }

.conn-card:active { transform: translateY(0); }

.conn-card-selected {
  border-color: color-mix(in srgb, var(--accent) 28%, var(--border-subtle));
  background: color-mix(in srgb, var(--accent) 2%, var(--surface));
}

.conn-card-selected::before {
  width: 3px;
  background: var(--accent);
}

.conn-selected-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--accent);
  flex-shrink: 0;
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 15%, transparent);
}

@keyframes connCardEnter {
  from { opacity: 0; transform: translateY(8px); }
  to   { opacity: 1; transform: translateY(0); }
}

/* ── Body ───────────────────────────────────────────────────── */
.conn-card-body {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 16px;
}

/* ── Head: Name (left) · Type · Source · Status (right) ────── */
.conn-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-width: 0;
}

.conn-head-left {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  flex: 1;
}

.conn-head-right {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.conn-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  letter-spacing: -0.01em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.conn-type {
  font-size: 11px;
  font-weight: 600;
  color: var(--type-color, var(--accent));
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.conn-dot {
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: var(--border-strong);
  flex-shrink: 0;
}

.conn-source {
  font-size: 11px;
  font-weight: 500;
  padding: 1px 6px;
  border-radius: 4px;
  white-space: nowrap;
}

.conn-source--local {
  color: var(--accent);
  background: var(--accent-soft);
}

.conn-source--remote {
  color: var(--warning);
  background: var(--warning-soft);
}

.conn-status {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 500;
  color: var(--text-secondary);
}

.conn-status[data-status='online']   { color: var(--success); }
.conn-status[data-status='offline']  { color: var(--danger); }

/* ── Address line ──────────────────────────────────────────── */
.conn-address {
  display: flex;
  align-items: center;
  gap: 4px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11.5px;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
}

.conn-host {
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--text-primary);
}

.conn-sep { color: var(--border-strong); }

.conn-user {
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ── Password row ──────────────────────────────────────────── */
.conn-password {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 9px;
  background: var(--muted);
  border-radius: 6px;
}

.conn-password-val {
  flex: 1;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11.5px;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.conn-password-val--masked {
  color: var(--text-tertiary);
  letter-spacing: 0.1em;
}

.conn-eye {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 4px;
  background: transparent;
  border: none;
  color: var(--text-tertiary);
  cursor: pointer;
  transition: all 180ms ease;
  flex-shrink: 0;
}

.conn-eye:hover {
  background: color-mix(in srgb, var(--type-color, var(--accent)) 10%, transparent);
  color: var(--type-color, var(--accent));
}

.conn-eye--on {
  background: color-mix(in srgb, var(--type-color, var(--accent)) 12%, transparent);
  color: var(--type-color, var(--accent));
}

/* ── Description ───────────────────────────────────────────── */
.conn-desc {
  margin: 0;
  font-size: 11.5px;
  line-height: 1.45;
  color: var(--text-secondary);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.conn-desc--empty {
  color: var(--text-tertiary);
  font-style: italic;
  opacity: 0.55;
}

/* ── Flip (details panel) ──────────────────────────────────── */
.conn-flip {
  perspective: 900px;
}

.conn-flip-inner {
  position: relative;
  width: 100%;
  min-height: 64px;
  transform-style: preserve-3d;
  transition: transform 500ms cubic-bezier(0.16, 1, 0.3, 1);
  cursor: pointer;
}

.conn-flip--on .conn-flip-inner {
  transform: rotateY(180deg);
}

.conn-flip-face {
  position: absolute;
  inset: 0;
  backface-visibility: hidden;
  -webkit-backface-visibility: hidden;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.conn-flip-back {
  transform: rotateY(180deg);
}

.conn-detail {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px 10px;
  padding: 6px 10px;
  background: var(--muted);
  border-radius: 6px;
}

.conn-detail-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  min-width: 0;
}

.conn-detail-label {
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-tertiary);
  font-weight: 500;
  flex-shrink: 0;
}

.conn-detail-val {
  font-size: 11px;
  color: var(--text-primary);
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

/* ── Duplicate tag ─────────────────────────────────────────── */
.conn-dup {
  align-self: flex-start;
}

.conn-dup span {
  display: inline-flex;
  align-items: center;
  padding: 2px 7px;
  border-radius: 4px;
  background: var(--warning-soft);
  color: var(--warning);
  font-size: 10.5px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

/* ── Footer ────────────────────────────────────────────────── */
.conn-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  padding-top: 4px;
}

.conn-footer-left,
.conn-footer-right {
  display: flex;
  align-items: center;
  gap: 0;
}

.conn-btn {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 3px 7px;
  border-radius: 4px;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all 180ms ease;
  white-space: nowrap;
  opacity: 0;
  transform: translateY(2px);
  pointer-events: none;
}

.conn-card:hover .conn-btn,
.conn-card-selected .conn-btn {
  opacity: 1;
  transform: translateY(0);
  pointer-events: auto;
}

.conn-btn:hover {
  background: color-mix(in srgb, var(--type-color, var(--accent)) 10%, transparent);
  color: var(--type-color, var(--accent));
}

.conn-btn:active { transform: scale(0.96); }

.conn-btn--danger:hover {
  background: color-mix(in srgb, var(--danger) 14%, transparent);
  color: var(--danger);
}

/* ── Responsive ────────────────────────────────────────────── */
@media (max-width: 640px) {
  .conn-card-body { padding: 12px 14px; }
  .conn-btn { opacity: 1; transform: none; pointer-events: auto; }
}

/* ── Reduced motion ────────────────────────────────────────── */
@media (prefers-reduced-motion: reduce) {
  .conn-card,
  .conn-btn {
    animation: none !important;
    transition: none !important;
  }
  .conn-card:hover { transform: none; }
}
</style>
