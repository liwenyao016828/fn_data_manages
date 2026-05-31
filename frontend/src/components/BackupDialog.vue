<template>
  <Dialog :open="modelValue" @update:open="$emit('update:modelValue', $event)">
    <DialogContent class="max-w-[800px]">
      <DialogHeader>
        <DialogTitle>{{ dialogTitle }}</DialogTitle>
      </DialogHeader>

      <template v-if="!importMode">
        <div class="flex items-center justify-between mb-4">
          <span class="flex items-center gap-3">
            <Badge variant="secondary" class="bg-emerald-500/10 text-emerald-600 border-emerald-200">
              {{ database.name }}
            </Badge>
            <span v-if="bakList.length" class="text-xs text-muted-foreground">共 {{ bakList.length }} 个备份</span>
          </span>
          <Button size="sm" @click="showCreate = true">
            <Plus class="h-4 w-4" />
            创建备份
          </Button>
        </div>

        <div v-if="bakList.length > 0" class="rounded-xl border border-border overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead class="min-w-[180px]">备份名称</TableHead>
                <TableHead class="w-[80px]">级别</TableHead>
                <TableHead class="w-[80px]">来源</TableHead>
                <TableHead class="w-[90px]">大小</TableHead>
                <TableHead class="min-w-[160px]">创建时间</TableHead>
                <TableHead class="text-center w-[200px]">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="row in bakList" :key="row.id">
                <TableCell>{{ row.name }}</TableCell>
                <TableCell>
                  <Badge v-if="row.backupLevel === 'system'" variant="outline" class="bg-amber-500/10 text-amber-600 border-amber-200">系统</Badge>
                  <Badge v-else-if="row.backupLevel === 'redis'" variant="outline" class="bg-amber-500/10 text-amber-600 border-amber-200">Redis</Badge>
                  <Badge v-else variant="outline" class="bg-blue-500/10 text-blue-600 border-blue-200">MySQL</Badge>
                </TableCell>
                <TableCell>
                  <Badge
                    :variant="row.backupType === 'import' ? 'secondary' : row.backupType === 'scheduled' ? 'outline' : 'default'"
                    class="text-xs"
                  >
                    {{ row.backupType === 'import' ? '导入' : row.backupType === 'scheduled' ? '定时' : '手动' }}
                  </Badge>
                </TableCell>
                <TableCell>{{ formatSize(row.fileSize) }}</TableCell>
                <TableCell>{{ formatLogTime(row.createdAt) }}</TableCell>
                <TableCell>
                  <div class="flex items-center gap-1">
                    <Button variant="link" size="sm" class="h-auto p-0 text-primary" @click="handleRestore(row)">恢复</Button>
                    <Button variant="link" size="sm" class="h-auto p-0 text-primary" @click="handleDownload(row)">下载</Button>
                    <Button variant="link" size="sm" class="h-auto p-0 text-destructive" @click="confirmDelete(row)">删除</Button>
                  </div>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
        <div v-else class="flex flex-col items-center justify-center py-10 text-muted-foreground">
          <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round" class="mb-3 opacity-30"><path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><polyline points="13 2 13 9 20 9"/></svg>
          <span class="text-sm">暂无备份记录</span>
        </div>

        <Dialog :open="showCreate" @update:open="showCreate = $event">
          <DialogContent class="max-w-[420px]">
            <DialogHeader>
              <DialogTitle>创建备份</DialogTitle>
            </DialogHeader>
            <div class="flex flex-col gap-4">
              <div class="flex flex-col gap-1.5">
                <label class="text-sm font-medium">备份名称</label>
                <Input v-model="createForm.name" placeholder="留空自动生成" />
              </div>
              <div class="flex flex-col gap-1.5">
                <label class="text-sm font-medium">描述</label>
                <Textarea v-model="createForm.description" :rows="2" placeholder="可选" />
              </div>
            </div>
            <div class="flex justify-end gap-2 mt-4">
              <Button variant="outline" @click="showCreate = false">取消</Button>
              <Button @click="submitCreate" :disabled="creating">
                <Loader2 v-if="creating" class="h-4 w-4 animate-spin" />
                确认
              </Button>
            </div>
          </DialogContent>
        </Dialog>
      </template>

      <template v-else>
        <div class="py-2.5 flex flex-col gap-4">
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-muted-foreground">目标数据库 <span class="text-red-500">*</span></label>
            <Select v-model="importDbName">
              <SelectTrigger class="w-full">
                <template v-if="selectedImportDb">
                  <span class="flex items-center gap-1.5 truncate">
                    {{ selectedImportDb.name }}
                    <Badge
                      :variant="selectedImportDb.type === 'redis' ? 'outline' : 'default'"
                      :class="selectedImportDb.type === 'redis' ? 'bg-amber-500/10 text-amber-600 border-amber-200' : 'bg-blue-500/10 text-blue-600 border-blue-200'"
                      class="text-[10px] px-1.5 py-0 shrink-0"
                    >
                      {{ selectedImportDb.type === 'redis' ? 'Redis' : 'MySQL' }}
                    </Badge>
                    <span class="text-xs text-muted-foreground shrink-0">{{ selectedImportDb.host || 'local' }}</span>
                  </span>
                </template>
                <SelectValue v-else placeholder="请选择要导入的数据库" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="db in availableDbs"
                  :key="db.isRemote ? 'r:' + db.id : 'l:' + db.id"
                  :value="db.name"
                >
                  <div class="flex items-center justify-between w-full gap-2">
                    <span class="flex-1 truncate">{{ db.name }}</span>
                    <div class="flex items-center gap-2 shrink-0">
                      <Badge
                        :variant="db.type === 'redis' ? 'outline' : 'default'"
                        :class="db.type === 'redis' ? 'bg-amber-500/10 text-amber-600 border-amber-200' : 'bg-blue-500/10 text-blue-600 border-blue-200'"
                        class="text-[10px] px-1.5 py-0"
                      >
                        {{ db.type === 'redis' ? 'Redis' : 'MySQL' }}
                      </Badge>
                      <span class="text-xs text-muted-foreground">{{ db.host || 'local' }}</span>
                    </div>
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div
            class="border-2 border-dashed border-input rounded-lg p-10 text-center cursor-pointer transition-colors hover:border-primary"
            @click="triggerUpload"
            @dragover.prevent
            @drop.prevent="handleDrop"
          >
            <input ref="fileInput" type="file" accept=".sql,.sql.gz,.gz,.tar.gz,.rdb,.json" class="hidden" @change="handleFileChange" />
            <Upload class="h-12 w-12 mx-auto text-muted-foreground/50" />
            <p class="text-sm text-muted-foreground mt-4 mb-1">将备份文件拖拽到此处，或者 <span class="text-primary cursor-pointer">点击上传</span></p>
            <p class="text-xs text-muted-foreground/70">支持 .sql, .sql.gz, .tar.gz, .rdb, .json 文件</p>
          </div>
          <div v-if="selectedFile" class="flex items-center gap-3 mt-4">
            <Badge variant="secondary" class="gap-1 pr-1">
              {{ selectedFile.name }} ({{ formatSize(selectedFile.size) }})
              <button class="ml-1 hover:text-destructive" @click="clearFile">
                <X class="h-3 w-3" />
              </button>
            </Badge>
            <Button size="sm" @click="doImport" :disabled="importing">
              <Loader2 v-if="importing" class="h-4 w-4 animate-spin" />
              确认导入
            </Button>
          </div>
        </div>
      </template>

      <div v-if="!importMode" class="flex justify-end mt-4">
        <Button variant="outline" @click="$emit('update:modelValue', false)">关闭</Button>
      </div>
    </DialogContent>
  </Dialog>

  <Dialog :open="showConfirmDialog" @update:open="showConfirmDialog = $event">
    <DialogContent class="max-w-[400px]">
      <DialogHeader>
        <DialogTitle>确认删除</DialogTitle>
        <DialogDescription>确定要删除此备份吗？此操作不可撤销。</DialogDescription>
      </DialogHeader>
      <div class="flex justify-end gap-2 mt-2">
        <Button variant="outline" @click="showConfirmDialog = false">取消</Button>
        <Button variant="destructive" @click="doDelete">确定删除</Button>
      </div>
    </DialogContent>
  </Dialog>

  <Dialog :open="showRestoreDialog" @update:open="showRestoreDialog = $event">
    <DialogContent class="max-w-[400px]">
      <DialogHeader>
        <DialogTitle>恢复确认</DialogTitle>
        <DialogDescription>确定从备份 "{{ restoreRow?.name }}" 恢复数据库 "{{ database?.name }}" 吗？恢复不可撤销！</DialogDescription>
      </DialogHeader>
      <div class="flex justify-end gap-2 mt-2">
        <Button variant="outline" @click="showRestoreDialog = false">取消</Button>
        <Button variant="destructive" @click="doRestore">确定恢复</Button>
      </div>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { toast } from 'vue-sonner'
import { Plus, Upload, X, Loader2 } from 'lucide-vue-next'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/Dialog.vue'
import { Button } from '@/components/ui/Button.vue'
import { Input } from '@/components/ui/Input.vue'
import { Textarea } from '@/components/ui/Textarea.vue'
import { Badge } from '@/components/ui/Badge.vue'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/Table.vue'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/Select.vue'
import { formatLogTime } from '@/lib/utils'

const props = defineProps({
  modelValue: Boolean,
  database: Object,
  importMode: { type: Boolean, default: false }
})

const emit = defineEmits(['update:modelValue'])

const bakList = ref([])
const creating = ref(false)
const importing = ref(false)
const showCreate = ref(false)
const selectedFile = ref(null)
const fileInput = ref(null)
const importDbName = ref(props.database?.name || '')
const availableDbs = ref([])
const showConfirmDialog = ref(false)
const deleteRow = ref(null)
const showRestoreDialog = ref(false)
const restoreRow = ref(null)

const createForm = ref({ name: '', description: '' })

const dialogTitle = computed(() => {
  if (props.importMode) return '导入备份'
  return `备份管理 - ${props.database?.name || ''}`
})

const selectedImportDb = computed(() => {
  return availableDbs.value.find(db => db.name === importDbName.value)
})

const formatSize = (bytes) => {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0, size = bytes
  while (size >= 1024 && i < units.length - 1) { size /= 1024; i++ }
  return size.toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
}

const loadBackups = () => {
  if (!props.database) return
  const params = new URLSearchParams()
  params.set('server_id', String(props.database.id || ''))
  if (props.database.isRemote) params.set('source', 'remote')
  fetch(`/api/backups?${params.toString()}`)
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) bakList.value = data.data || []
    })
    .catch((e) => { console.error(e) })
}

watch(() => props.modelValue, (val) => {
  if (val) {
    if (props.importMode) {
      loadAvailableDatabases()
    }
    loadBackups()
  }
})

const loadAvailableDatabases = () => {
  Promise.all([
    fetch('/api/databases/db/list/all').then(r => r.json()),
    fetch('/api/remote-servers').then(r => r.json())
  ]).then(([localRes, remoteRes]) => {
    const locals = (localRes.code === 0 ? localRes.data : []) || []
    const remotes = (remoteRes.code === 0 ? remoteRes.data : []) || []
    availableDbs.value = [...locals, ...remotes.map(r => ({ ...r, isRemote: true }))]
  }).catch((e) => { console.error(e) })
}

const submitCreate = () => {
  creating.value = true
  const dbType = props.database?.type || 'mysql'
  const backupLevel = dbType === 'redis' ? 'redis' : 'mysql'
  fetch('/api/backups', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name: createForm.value.name || '',
      description: createForm.value.description,
      backupLevel: backupLevel,
      database: props.database?.database || props.database?.name || '',
      serverId: props.database?.id || 0,
      type: dbType,
      host: props.database?.host || '',
      port: props.database?.port || (dbType === 'redis' ? 6379 : 3306)
    })
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        toast.success('备份创建成功')
        showCreate.value = false
        loadBackups()
      } else {
        toast.error(data.msg || '创建失败')
      }
    })
    .catch(() => toast.error('创建失败'))
    .finally(() => { creating.value = false })
}

const handleDownload = (row) => {
  window.open(`/api/backups/${row.id}`, '_blank')
}

const confirmDelete = (row) => {
  deleteRow.value = row
  showConfirmDialog.value = true
}

const doDelete = () => {
  const row = deleteRow.value
  if (!row) return
  fetch(`/api/backups/${row.id}`, { method: 'DELETE' })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) { toast.success('删除成功'); loadBackups() }
      else toast.error(data.msg || '删除失败')
    })
    .catch(() => toast.error('删除失败'))
    .finally(() => { showConfirmDialog.value = false })
}

const handleRestore = (row) => {
  restoreRow.value = row
  showRestoreDialog.value = true
}

const doRestore = () => {
  const row = restoreRow.value
  if (!row) return
  const source = row.source || (props.database?.isRemote ? 'remote' : 'local')
  const isRedis = row.backupLevel === 'redis'
  const endpoint = isRedis ? `/api/redis/restore?source=${source}` : '/api/backups/restore'
  fetch(endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ backup_id: row.id, source })
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        toast.success(data.msg || '恢复成功')
        loadBackups()
      } else {
        toast.error(data.msg || '恢复失败')
      }
    })
    .catch(err => toast.error('恢复失败: ' + err.message))
    .finally(() => { showRestoreDialog.value = false })
}

const triggerUpload = () => { fileInput.value?.click() }
const handleFileChange = (e) => {
  const file = e.target.files?.[0]
  if (file) selectedFile.value = file
}
const handleDrop = (e) => {
  e.preventDefault()
  const file = e.dataTransfer?.files?.[0]
  if (file) {
    const ext = file.name.toLowerCase()
    if (!ext.endsWith('.sql') && !ext.endsWith('.gz') && !ext.endsWith('.tar.gz') && !ext.endsWith('.rdb') && !ext.endsWith('.json')) {
      toast.warning('仅支持 .sql, .sql.gz, .tar.gz, .rdb, .json 格式的文件')
      return
    }
    selectedFile.value = file
  }
}
const clearFile = () => {
  selectedFile.value = null
  if (fileInput.value) fileInput.value.value = ''
}

const doImport = () => {
  if (!selectedFile.value) { toast.warning('请选择备份文件'); return }
  const dbName = importDbName.value || props.database?.database || props.database?.name
  if (!dbName) { toast.warning('请选择目标数据库'); return }
  const selectedDb = availableDbs.value.find(d => d.name === dbName || d.database === dbName)
  const dbType = selectedDb?.type || props.database?.type || 'mysql'
  importing.value = true
  const fd = new FormData()
  fd.append('file', selectedFile.value)
  fd.append('name', dbName)
  fd.append('database', selectedDb?.database || dbName)
  fd.append('serverId', String(selectedDb?.id || props.database?.id || 0))
  fd.append('type', dbType)
  const fileName = selectedFile.value.name.toLowerCase()
  const isSystemFile = fileName.endsWith('.json') && dbType !== 'redis'
  fd.append('backupLevel', isSystemFile ? 'system' : dbType)
  if (selectedDb?.isRemote) fd.append('source', 'remote')
  fetch('/api/backups/import', { method: 'POST', body: fd })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        toast.success('导入成功')
        clearFile()
        loadBackups()
      } else toast.error(data.msg || '导入失败')
    })
    .catch(() => toast.error('导入失败'))
    .finally(() => { importing.value = false })
}
</script>
