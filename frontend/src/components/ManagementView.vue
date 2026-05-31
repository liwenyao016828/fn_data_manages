<template>
  <div class="page-padding h-full overflow-y-auto">
    <div class="content-card">
      <div class="content-header">
        <div class="flex items-start justify-between">
          <div>
            <h2 class="text-[15px] font-semibold text-foreground">数据库连接</h2>
            <p class="text-[13px] text-muted-foreground mt-0.5">管理所有数据库实例连接</p>
          </div>
          <div class="flex items-center gap-2 flex-wrap">
            <div class="flex h-[32px] items-center rounded-lg border border-border bg-white px-2 gap-1">
              <Search class="h-3.5 w-3.5 text-muted-foreground" />
              <Input v-model="searchKeyword" placeholder="搜索实例..." class="border-0 shadow-none h-[28px] text-[13px] w-[180px] bg-transparent" @input="handleSearch" />
            </div>
            <Button variant="outline" size="sm" class="h-[32px] text-[13px]" @click="showDetectDialog = true">
              <Search class="h-3.5 w-3.5 mr-1.5" />自动检测
            </Button>
            <Button variant="primary" size="sm" class="h-[32px] text-[13px]" @click="showCreateDialog = true">
              <Plus class="h-3.5 w-3.5 mr-1.5" />新建连接
            </Button>
          </div>
        </div>
      </div>

      <!-- Stats -->
      <div class="grid grid-cols-2 md:grid-cols-4 gap-3 content-body">
        <div class="flex items-center gap-3 rounded-lg bg-muted p-3">
          <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-500">
            <Database class="h-4 w-4" />
          </div>
          <div class="flex flex-col">
            <span class="text-lg font-bold text-foreground leading-tight">{{ stats.mysql }}</span>
            <span class="text-[11px] text-muted-foreground">MySQL</span>
          </div>
        </div>
        <div class="flex items-center gap-3 rounded-lg bg-muted p-3">
          <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-orange-50 text-orange-500">
            <Server class="h-4 w-4" />
          </div>
          <div class="flex flex-col">
            <span class="text-lg font-bold text-foreground leading-tight">{{ stats.redis }}</span>
            <span class="text-[11px] text-muted-foreground">Redis</span>
          </div>
        </div>
        <div class="flex items-center gap-3 rounded-lg bg-muted p-3">
          <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-emerald-50 text-emerald-500">
            <BarChart3 class="h-4 w-4" />
          </div>
          <div class="flex flex-col">
            <span class="text-lg font-bold text-foreground leading-tight">{{ stats.total }}</span>
            <span class="text-[11px] text-muted-foreground">总计</span>
          </div>
        </div>
        <div class="flex items-center gap-3 rounded-lg bg-muted p-3">
          <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-500">
            <Activity class="h-4 w-4" />
          </div>
          <div class="flex flex-col">
            <span class="text-lg font-bold text-foreground leading-tight">{{ stats.running }}</span>
            <span class="text-[11px] text-muted-foreground">运行中</span>
          </div>
        </div>
      </div>

      <!-- Tabs -->
      <div class="px-5 pb-3">
        <Tabs v-model="activeTab" class="w-full">
          <TabsList class="bg-muted">
            <TabsTrigger value="all" class="text-[12px] data-[state=active]:bg-white data-[state=active]:text-foreground">所有实例</TabsTrigger>
            <TabsTrigger value="mysql" class="text-[12px] data-[state=active]:bg-white data-[state=active]:text-foreground">MySQL</TabsTrigger>
            <TabsTrigger value="redis" class="text-[12px] data-[state=active]:bg-white data-[state=active]:text-foreground">Redis</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      <!-- Table -->
      <div class="border-t border-border">
        <Table class="w-full">
          <colgroup>
            <col class="w-8" />
            <col />
            <col class="tbl-col-type" />
            <col class="tbl-col-host" />
            <col class="tbl-col-user" />
            <col class="tbl-col-ver" />
            <col class="tbl-col-disk" />
            <col class="tbl-col-status" />
            <col class="tbl-col-action" />
          </colgroup>
          <TableHeader>
            <TableRow class="hover:bg-transparent border-b border-border">
              <TableHead class="text-[12px] font-normal text-muted-foreground h-10" />
              <TableHead class="text-[12px] font-normal text-muted-foreground h-10">名称</TableHead>
              <TableHead class="text-[12px] font-normal text-muted-foreground h-10">类型</TableHead>
              <TableHead class="text-[12px] font-normal text-muted-foreground h-10">主机</TableHead>
              <TableHead class="text-[12px] font-normal text-muted-foreground h-10">用户名</TableHead>
              <TableHead class="text-[12px] font-normal text-muted-foreground h-10">版本</TableHead>
              <TableHead class="text-[12px] font-normal text-muted-foreground h-10">磁盘</TableHead>
              <TableHead class="text-[12px] font-normal text-muted-foreground h-10">状态</TableHead>
              <TableHead class="text-center text-[12px] font-normal text-muted-foreground h-10">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <template v-if="tableData.length === 0">
              <TableRow>
                <TableCell colspan="10" class="h-64 text-center">
                  <div class="flex flex-col items-center justify-center text-muted-foreground">
                    <Inbox class="h-10 w-10 mb-2 opacity-30" />
                    <p class="text-[13px]">暂无数据</p>
                  </div>
                </TableCell>
              </TableRow>
            </template>
            <template v-for="row in tableData" :key="instanceUid(row)">
              <TableRow class="cursor-pointer transition-colors duration-150 border-b border-[#F0F0F0]" :class="[connectionId === instanceUid(row) ? 'bg-primary/[0.04] hover:bg-primary/[0.08]' : 'hover:bg-muted']" @click="toggleExpand(row)">
                <TableCell class="w-8">
                  <ChevronRight class="h-4 w-4 text-muted-foreground transition-transform duration-200" :class="{ 'rotate-90': expandedRowUid === instanceUid(row) }" />
                </TableCell>
                <TableCell>
                  <div class="flex items-center gap-2 min-w-0">
                    <StatusDot :status="connectionId === instanceUid(row) ? 'selected' : (onlineStatus[instanceUid(row)] ? 'online' : 'default')" />
                    <span class="text-[13px] text-foreground truncate">{{ row.name }}</span>
                    <Badge :class="row.isRemote ? 'bg-orange-50 text-orange-500 border-orange-200' : 'bg-blue-50 text-blue-500 border-blue-200'" variant="outline" class="text-[10px] h-[18px] ml-0.5 shrink-0">
                      {{ row.isRemote ? '远程' : '本地' }}
                    </Badge>
                    <Badge v-if="isDuplicate(row)" variant="outline" class="bg-amber-50 text-amber-600 border-amber-200 text-[10px] h-[18px] shrink-0">
                      重复
                    </Badge>
                  </div>
                </TableCell>
                <TableCell>
                  <Badge :variant="row.type === 'mysql' ? 'default' : 'secondary'" class="rounded-full text-[10px] py-0">
                    {{ row.type === 'mysql' ? 'MySQL' : 'Redis' }}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div class="flex items-center gap-1 min-w-0">
                    <span class="text-[13px] text-secondary-foreground truncate">{{ row.host }}</span>
                    <span class="text-[11px] text-muted-foreground shrink-0">:{{ row.port }}</span>
                  </div>
                </TableCell>
                <TableCell><span class="text-[13px] text-muted-foreground truncate block">{{ row.username }}</span></TableCell>
                <TableCell><span class="text-[13px] text-muted-foreground truncate block">{{ row.version || '-' }}</span></TableCell>
                <TableCell><span class="text-[13px] text-muted-foreground truncate block">{{ row.disk || '-' }}</span></TableCell>
                <TableCell>
                  <div class="flex items-center gap-1.5">
                    <StatusDot :status="onlineStatus[instanceUid(row)] ? 'online' : 'offline'" size="sm" />
                    <span class="text-[12px]" :class="onlineStatus[instanceUid(row)] ? 'text-emerald-600' : 'text-muted-foreground'">{{ onlineStatus[instanceUid(row)] ? '在线' : '离线' }}</span>
                  </div>
                </TableCell>
                <TableCell @click.stop>
                  <div class="flex items-center gap-0.5 shrink-0">
                    <Button variant="ghost" size="xs" class="text-muted-foreground hover:bg-muted min-w-[32px]" @click="handleEdit(row)">编辑</Button>
                    <Button variant="ghost" size="xs" class="text-destructive hover:bg-destructive/10 min-w-[32px]" @click="handleDelete(row)">删除</Button>
                  </div>
                </TableCell>
              </TableRow>
              <TableRow v-if="expandedRowUid === instanceUid(row)" @click.stop>
                <TableCell colspan="10" class="!p-0">
                  <div class="expand-row">
                    <div class="expand-row-inner">
                      <div class="expand-fields">
                        <div class="expand-field">
                          <span class="expand-label">描述</span>
                          <span class="expand-value">{{ row.description || '无' }}</span>
                        </div>
                        <div class="expand-field">
                          <span class="expand-label">权限</span>
                          <span class="expand-value">{{ row.permission || '%' }}</span>
                        </div>
                        <div class="expand-field">
                          <span class="expand-label">SSL</span>
                          <span class="expand-value">{{ row.ssl ? '开启' : '关闭' }}</span>
                        </div>
                        <div class="expand-field">
                          <span class="expand-label">创建时间</span>
                          <span class="expand-value">{{ formatLogTime(row.createdAt) || '-' }}</span>
                        </div>
                      </div>
                      <div class="expand-actions">
                        <Button variant="outline" size="sm" class="h-[28px] text-[12px]" @click="navigateTo('data', row)">数据管理</Button>
                        <Button variant="outline" size="sm" class="h-[28px] text-[12px]" @click="navigateTo('backup', row)">备份列表</Button>
                        <Button variant="outline" size="sm" class="h-[28px] text-[12px]" @click="navigateTo('logs', row)" v-if="logEnabled">查看日志</Button>
                      </div>
                    </div>
                  </div>
                </TableCell>
              </TableRow>
            </template>
          </TableBody>
        </Table>
      </div>

      <!-- Pagination -->
      <div class="flex items-center justify-between section-padding border-t border-border flex-wrap gap-2">
        <span class="text-[12px] text-muted-foreground">共 {{ total }} 条</span>
        <div class="flex items-center gap-2">
          <Select v-model="pageSize" @update:model-value="onPageSizeChange">
            <SelectTrigger class="h-[32px] w-[70px] text-[12px] border-border shadow-none">
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
              <ChevronLeft class="h-3.5 w-3.5 text-muted-foreground" />
            </Button>
            <span class="text-[12px] px-2 text-muted-foreground">{{ page }} / {{ totalPages }}</span>
            <Button variant="outline" size="icon" class="h-[32px] w-[32px]" :disabled="page >= totalPages" @click="page++">
              <ChevronRight class="h-3.5 w-3.5 text-muted-foreground" />
            </Button>
          </div>
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
          <DialogDescription class="text-[13px] text-muted-foreground">
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

const toggleExpand = (row) => {
  const uid = instanceUid(row)
  expandedRowUid.value = expandedRowUid.value === uid ? null : uid
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
    allData.value = [...local, ...remote]
    stats.value.total = allData.value.length
    stats.value.mysql = allData.value.filter(i => (i.type || 'mysql') === 'mysql').length
    stats.value.redis = allData.value.filter(i => i.type === 'redis').length
    stats.value.running = Object.values(onlineStatus.value).filter(Boolean).length
    autoRefreshMissing(allData.value)
    checkOnlineStatus(allData.value)
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
.tbl-col-type { width: 68px; min-width: 56px; }
.tbl-col-host { width: 15%; min-width: 120px; }
.tbl-col-user { width: 10%; min-width: 72px; }
.tbl-col-ver { width: 9%; min-width: 64px; }
.tbl-col-disk { width: 8%; min-width: 52px; }
.tbl-col-status { width: 68px; min-width: 56px; }
.tbl-col-action { width: 88px; min-width: 72px; }

.expand-row {
  background: var(--muted);
  padding: 12px 20px 12px 44px;
}

.expand-row-inner {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 12px 24px;
}

.expand-fields {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 24px;
  flex: 1;
  min-width: 0;
}

.expand-field {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 100px;
}

.expand-label {
  font-size: 11px;
  color: var(--muted-foreground);
}

.expand-value {
  font-size: 13px;
  color: var(--foreground);
  word-break: break-all;
}

.expand-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

@media (max-width: 640px) {
  .expand-row {
    padding: 10px 12px 10px 28px;
  }
  .expand-fields {
    gap: 6px 16px;
  }
  .expand-field {
    min-width: 80px;
  }
  .expand-actions {
    width: 100%;
    flex-wrap: wrap;
  }
}
</style>
