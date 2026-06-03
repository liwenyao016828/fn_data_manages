<template>
  <Dialog :open="modelValue" @update:open="$emit('update:modelValue', $event)">
    <DialogContent class="max-w-[800px]">
      <DialogHeader>
        <DialogTitle>{{ dialogTitle }}</DialogTitle>
      </DialogHeader>

      <template v-if="!importMode">
        <div class="flex items-center justify-between mb-4">
          <span class="flex items-center gap-3">
            <span class="badge-status badge-status-success text-[11px]">
              {{ database.name }}
            </span>
            <span v-if="bakList.length" class="text-[11px]" style="color: var(--text-tertiary)">共 {{ bakList.length }} 个备份</span>
          </span>
          <button class="btn-primary" @click="showCreate = true">
            <Plus class="h-4 w-4" />
            创建备份
          </button>
        </div>

        <div v-if="bakList.length > 0" class="flex flex-col gap-2">
          <div
            v-for="row in bakList"
            :key="row.id"
            class="rounded-xl p-3.5 flex items-center gap-3 transition-all duration-200"
            style="background: var(--surface); border: 1px solid var(--border-subtle)"
          >
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <span class="text-[13px] font-medium truncate" style="color: var(--text-primary)">{{ row.name }}</span>
                <span
                  class="badge-status text-[10px]"
                  :class="row.backupLevel === 'system' || row.backupLevel === 'redis' ? 'badge-status-warning' : 'badge-status-info'"
                >
                  {{ row.backupLevel === 'system' ? '系统' : row.backupLevel === 'redis' ? 'Redis' : 'MySQL' }}
                </span>
                <span
                  class="badge-status text-[10px]"
                  :class="row.backupType === 'import' ? 'badge-status-neutral' : row.backupType === 'scheduled' ? 'badge-status-warning' : 'badge-status-info'"
                >
                  {{ row.backupType === 'import' ? '导入' : row.backupType === 'scheduled' ? '定时' : '手动' }}
                </span>
              </div>
              <div class="flex items-center gap-3 mt-1">
                <span class="text-[11px]" style="color: var(--text-tertiary)">{{ formatSize(row.fileSize) }}</span>
                <span class="text-[11px]" style="color: var(--text-tertiary)">{{ formatLogTime(row.createdAt) }}</span>
              </div>
            </div>
            <div class="flex items-center gap-1 shrink-0">
              <button class="btn-ghost" @click="handleRestore(row)">恢复</button>
              <button class="btn-ghost" @click="handleDownload(row)">下载</button>
              <button class="btn-ghost-danger" @click="confirmDelete(row)">删除</button>
            </div>
          </div>
        </div>
        <div v-else class="empty-state py-10">
          <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round" class="empty-state-icon"><path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><polyline points="13 2 13 9 20 9"/></svg>
          <span class="empty-state-text">暂无备份记录</span>
        </div>

        <Dialog :open="showCreate" @update:open="showCreate = $event">
          <DialogContent class="max-w-[420px]">
            <DialogHeader>
              <DialogTitle>创建备份</DialogTitle>
            </DialogHeader>
            <div class="flex flex-col gap-4">
              <div class="flex flex-col gap-1.5">
                <label class="text-[12px] font-medium" style="color: var(--text-secondary)">备份名称</label>
                <Input v-model="createForm.name" placeholder="留空自动生成" />
              </div>
              <div class="flex flex-col gap-1.5">
                <label class="text-[12px] font-medium" style="color: var(--text-secondary)">描述</label>
                <Textarea v-model="createForm.description" :rows="2" placeholder="可选" />
              </div>
            </div>
            <div class="flex justify-end gap-2 mt-4">
              <button class="btn-ghost" @click="showCreate = false">取消</button>
              <button class="btn-primary" @click="submitCreate" :disabled="creating">
                <Loader2 v-if="creating" class="h-4 w-4 animate-spin" />
                确认
              </button>
            </div>
          </DialogContent>
        </Dialog>
      </template>

      <template v-else>
        <div class="py-2.5 flex flex-col gap-4">
          <div class="flex flex-col gap-1.5">
            <label class="text-[12px] font-medium" style="color: var(--text-secondary)">目标数据库 <span style="color: var(--danger)">*</span></label>
            <Select v-model="importDbName">
              <SelectTrigger class="w-full">
                <template v-if="selectedImportDb">
                  <span class="flex items-center gap-1.5 truncate">
                    {{ selectedImportDb.name }}
                    <span
                      class="badge-status text-[10px] px-1.5 py-0 shrink-0"
                      :class="selectedImportDb.type === 'redis' ? 'badge-status-warning' : 'badge-status-info'"
                    >
                      {{ selectedImportDb.type === 'redis' ? 'Redis' : 'MySQL' }}
                    </span>
                    <span class="text-[11px] shrink-0" style="color: var(--text-tertiary)">{{ selectedImportDb.host || 'local' }}</span>
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
                      <span
                        class="badge-status text-[10px] px-1.5 py-0"
                        :class="db.type === 'redis' ? 'badge-status-warning' : 'badge-status-info'"
                      >
                        {{ db.type === 'redis' ? 'Redis' : 'MySQL' }}
                      </span>
                      <span class="text-[11px]" style="color: var(--text-tertiary)">{{ db.host || 'local' }}</span>
                    </div>
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div
            class="rounded-xl p-10 text-center cursor-pointer transition-all duration-200"
            style="border: 2px dashed var(--border); background: var(--surface)"
            @click="triggerUpload"
            @dragover.prevent
            @drop.prevent="handleDrop"
          >
            <input ref="fileInput" type="file" accept=".sql,.sql.gz,.gz,.tar.gz,.rdb,.json" class="hidden" @change="handleFileChange" />
            <Upload class="h-12 w-12 mx-auto" style="color: var(--text-tertiary); opacity: 0.5" />
            <p class="text-[13px] mt-4 mb-1" style="color: var(--text-secondary)">将备份文件拖拽到此处，或者 <span style="color: var(--accent); cursor: pointer">点击上传</span></p>
            <p class="text-[11px]" style="color: var(--text-tertiary); opacity: 0.7">支持 .sql, .sql.gz, .tar.gz, .rdb, .json 文件</p>
          </div>
          <div v-if="selectedFile" class="flex items-center gap-3 mt-4">
            <span class="badge-status badge-status-neutral gap-1 pr-1">
              {{ selectedFile.name }} ({{ formatSize(selectedFile.size) }})
              <button class="ml-1" style="color: var(--danger)" @click="clearFile">
                <X class="h-3 w-3" />
              </button>
            </span>
            <button class="btn-primary text-[11px] h-7" @click="doImport" :disabled="importing">
              <Loader2 v-if="importing" class="h-4 w-4 animate-spin" />
              确认导入
            </button>
          </div>
        </div>
      </template>

      <div v-if="!importMode" class="flex justify-end mt-4">
        <button class="btn-secondary" @click="$emit('update:modelValue', false)">关闭</button>
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
        <button class="btn-ghost" @click="showConfirmDialog = false">取消</button>
        <button class="btn-danger" @click="doDelete">确定删除</button>
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
        <button class="btn-ghost" @click="showRestoreDialog = false">取消</button>
        <button class="btn-danger" @click="doRestore">确定恢复</button>
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
