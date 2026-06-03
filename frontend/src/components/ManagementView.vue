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

    <!-- Stats Row -->
    <div class="grid grid-cols-3 gap-3 section-gap">
      <div class="stat-card">
        <div class="stat-icon" style="background: var(--accent-soft); color: var(--accent)">
          <Database class="h-4 w-4" />
        </div>
        <div class="flex flex-col">
          <span class="stat-value">{{ stats.mysql }}</span>
          <span class="stat-label">MySQL</span>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background: var(--warning-soft); color: var(--warning)">
          <Server class="h-4 w-4" />
        </div>
        <div class="flex flex-col">
          <span class="stat-value">{{ stats.redis }}</span>
          <span class="stat-label">Redis</span>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background: var(--success-soft); color: var(--success)">
          <BarChart3 class="h-4 w-4" />
        </div>
        <div class="flex flex-col">
          <span class="stat-value">{{ stats.total }}</span>
          <span class="stat-label">总计</span>
        </div>
      </div>
    </div>

    <!-- Filter Bar -->
    <div class="flex items-center gap-3 section-gap flex-wrap">
      <div class="flex items-center rounded-xl border px-3 gap-1.5 h-[34px]" style="border-color: var(--border); background: var(--surface)">
        <Search class="h-3.5 w-3.5 shrink-0" style="color: var(--text-tertiary)" />
        <Input v-model="searchKeyword" placeholder="搜索实例..." class="border-0 shadow-none h-[28px] text-[13px] w-[180px] bg-transparent" @input="handleSearch" />
      </div>
      <div class="flex items-center gap-1.5">
        <span
          :class="['pill', activeTab === 'all' ? 'pill-active' : 'pill-default']"
          @click="activeTab = 'all'"
        >全部</span>
        <span
          :class="['pill', activeTab === 'mysql' ? 'pill-active' : 'pill-default']"
          @click="activeTab = 'mysql'"
        >MySQL</span>
        <span
          :class="['pill', activeTab === 'redis' ? 'pill-active' : 'pill-default']"
          @click="activeTab = 'redis'"
        >Redis</span>
      </div>
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
        v-for="row in tableData"
        :key="instanceUid(row)"
        class="content-card-interactive conn-card"
        :class="{ 'conn-card-selected': connectionId === instanceUid(row) }"
        @click="selectConnection(row)"
      >
        <!-- Top: Name + Type Badge + Status Dot -->
        <div class="flex items-center justify-between gap-2 mb-2.5">
          <div class="flex items-center gap-2 min-w-0">
            <StatusDot :status="connectionId === instanceUid(row) ? 'selected' : (onlineStatus[instanceUid(row)] ? 'online' : 'default')" />
            <span class="text-[13px] font-semibold truncate" style="color: var(--text-primary)">{{ row.name }}</span>
            <span
              class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium shrink-0"
              :style="{
                background: row.type === 'mysql' ? 'var(--accent-soft)' : 'var(--warning-soft)',
                color: row.type === 'mysql' ? 'var(--accent)' : 'var(--warning)',
              }"
            >{{ row.type === 'mysql' ? 'MySQL' : 'Redis' }}</span>
            <span
              v-if="row.isRemote"
              class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium shrink-0"
              style="background: var(--warning-soft); color: var(--warning)"
            >远程</span>
            <span
              v-else
              class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium shrink-0"
              style="background: var(--accent-soft); color: var(--accent)"
            >本地</span>
          </div>
          <div class="flex items-center gap-1.5 shrink-0">
            <StatusDot :status="onlineStatus[instanceUid(row)] ? 'online' : 'offline'" size="xs" />
            <span class="text-[11px]" :style="{ color: onlineStatus[instanceUid(row)] ? 'var(--success)' : 'var(--text-tertiary)' }">
              {{ onlineStatus[instanceUid(row)] ? '在线' : '离线' }}
            </span>
          </div>
        </div>

        <!-- Middle: Host:port, username, description -->
        <div class="flex flex-col gap-1 mb-1">
          <div class="flex items-center gap-1 min-w-0">
            <span class="text-[12px] font-mono-data truncate" style="color: var(--text-secondary)">{{ row.host }}</span>
            <span class="text-[11px] font-mono-data shrink-0" style="color: var(--text-tertiary)">:{{ row.port }}</span>
          </div>
          <span class="text-[12px] truncate" style="color: var(--text-tertiary)">{{ row.username }}</span>
          <span v-if="row.description" class="text-[12px] truncate" style="color: var(--text-tertiary)">{{ row.description }}</span>
        </div>

        <!-- Duplicate Badge -->
        <div v-if="isDuplicate(row)" class="mb-1.5">
          <span class="inline-flex items-center rounded-lg px-2 py-0.5 text-[10px] font-medium" style="background: var(--warning-soft); color: var(--warning); border: 1px solid color-mix(in srgb, var(--warning) 20%, transparent)">
            重复连接
          </span>
        </div>

        <!-- Bottom: Action buttons (shown on hover) -->
        <div class="conn-card-actions mt-auto pt-2">
          <div class="flex items-center gap-1.5 flex-wrap">
            <button class="btn-ghost" @click.stop="navigateTo('data', row)">数据</button>
            <button class="btn-ghost" @click.stop="navigateTo('backup', row)">备份</button>
            <button v-if="logEnabled" class="btn-ghost" @click.stop="navigateTo('logs', row)">日志</button>
            <div class="flex-1" />
            <button class="btn-ghost" @click.stop="handleEdit(row)">编辑</button>
            <button class="btn-ghost-danger" @click.stop="handleDelete(row)">删除</button>
          </div>
        </div>

        <!-- Expanded Detail Overlay (right half, shown on hover) -->
        <div class="conn-card-overlay" @click.stop="selectConnection(row)">
          <div class="flex flex-wrap gap-x-5 gap-y-2 w-full">
              <div class="flex flex-col gap-0.5">
                <span class="text-[11px]" style="color: var(--text-tertiary)">描述</span>
                <span class="text-[12px]" style="color: var(--text-primary)">{{ row.description || '无' }}</span>
              </div>
              <div class="flex flex-col gap-0.5">
                <span class="text-[11px]" style="color: var(--text-tertiary)">权限</span>
                <span class="text-[12px]" style="color: var(--text-primary)">{{ row.permission || '%' }}</span>
              </div>
              <div class="flex flex-col gap-0.5">
                <span class="text-[11px]" style="color: var(--text-tertiary)">SSL</span>
                <span class="text-[12px]" style="color: var(--text-primary)">{{ row.ssl ? '开启' : '关闭' }}</span>
              </div>
              <div class="flex flex-col gap-0.5">
                <span class="text-[11px]" style="color: var(--text-tertiary)">版本</span>
                <span class="text-[12px]" style="color: var(--text-primary)">{{ row.version || '-' }}</span>
              </div>
              <div class="flex flex-col gap-0.5">
                <span class="text-[11px]" style="color: var(--text-tertiary)">磁盘</span>
                <span class="text-[12px]" style="color: var(--text-primary)">{{ row.disk || '-' }}</span>
              </div>
              <div class="flex flex-col gap-0.5">
                <span class="text-[11px]" style="color: var(--text-tertiary)">创建时间</span>
                <span class="text-[12px]" style="color: var(--text-primary)">{{ formatLogTime(row.createdAt) || '-' }}</span>
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
import { ref, computed, onMounted, onActivated, watch, inject } from 'vue'
import { storeToRefs } from 'pinia'
import { useAppContext } from '../stores/context'
import { useHealthStore } from '../stores/health'
import { sourceParam, instanceUid } from '@/lib/instance'
import { formatLogTime } from '@/lib/utils'
import { toast } from 'vue-sonner'
import { Search, Plus, Database, Server, BarChart3, Activity, ChevronRight, ChevronLeft, Inbox } from 'lucide-vue-next'
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
const stats = ref({ mysql: 0, redis: 0, total: 0, running: 0 })
const healthStore = useHealthStore()
const { statusMap: onlineStatus } = storeToRefs(healthStore)
const previousIds = ref(new Set())
let searchTimer = null

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
    // 构建唯一键：主机:端口:用户名（全部小写）
    const key = `${(item.host || '').toLowerCase()}:${item.port}:${(item.username || '').toLowerCase()}`

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

const resetPageAndCollapse = () => {
  page.value = 1
  expandedRowUid.value = null
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
    stats.value.redis = allData.value.filter(i => i.type === 'redis').length
    stats.value.running = Object.values(onlineStatus.value).filter(Boolean).length
    autoRefreshMissing(allData.value)
    checkOnlineStatus(allData.value)
    newInstanceIds.forEach(uid => {
      healthStore.forceCheckOne(uid)
    })
    completeProgress?.()
  })
}

const checkOnlineStatus = async (items) => {
  await healthStore.refreshAll()
  stats.value.running = Object.values(onlineStatus.value).filter(Boolean).length
}

const autoRefreshMissing = (items) => {
  items.filter(item => !item.isRemote && (!item.version || !item.disk)).forEach(item => {
    fetch('/api/databases/refresh', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: item.id }) })
      .then(r => r.json()).then(data => {
        if (data.code === 0) {
          const idx = allData.value.findIndex(r => r.id === item.id && !r.isRemote)
          if (idx >= 0) {
            if (data.data.version && !allData.value[idx].version) allData.value[idx].version = data.data.version
            if (data.data.disk && !allData.value[idx].disk) allData.value[idx].disk = data.data.disk
          }
        }
      }).catch(() => {})
  })
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
onActivated(() => { fetchData() })
</script>

<style scoped>
.conn-card {
  padding: 0.75rem 1rem;
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 120px;
}

.conn-card::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  background: transparent;
  border-radius: 0 3px 3px 0;
  transition: background var(--transition-normal);
}

.conn-card-selected::before {
  background: var(--accent);
}

.conn-card-selected {
  border-color: color-mix(in srgb, var(--accent) 25%, var(--border));
  background: color-mix(in srgb, var(--accent) 3%, var(--surface));
}

.conn-card-actions {
  opacity: 0;
  transform: translateY(4px);
  transition: opacity var(--transition-normal), transform var(--transition-normal);
}

.conn-card:hover .conn-card-actions,
.conn-card-selected .conn-card-actions {
  opacity: 1;
  transform: translateY(0);
}

.conn-card-overlay {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 50%;
  padding: 0.75rem 1rem;
  background: color-mix(in srgb, var(--surface) 94%, transparent);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-radius: inherit;
  border-top-left-radius: 0;
  border-bottom-left-radius: 0;
  z-index: 5;
  cursor: pointer;
  display: flex;
  align-items: center;
  opacity: 0;
  transform: translateX(20px);
  pointer-events: none;
  transition: opacity 0.25s ease-out, transform 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.conn-card:hover .conn-card-overlay {
  opacity: 1;
  transform: translateX(0);
  pointer-events: auto;
  transition-delay: 0.3s;
}

.fade-slide-in {
  animation: fadeSlideIn 0.3s ease-out;
}

@keyframes fadeSlideIn {
  from {
    opacity: 0;
    transform: translateY(-8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 640px) {
  .conn-card-actions {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
