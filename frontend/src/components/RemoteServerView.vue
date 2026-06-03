<template>
  <div class="page-padding h-full overflow-y-auto">
    <!-- Page Header -->
    <div class="flex items-start justify-between mb-6">
      <div>
        <h2 class="text-[15px] font-semibold" style="color: var(--text-primary)">远程服务器</h2>
        <p class="text-[12px] mt-0.5" style="color: var(--text-tertiary)">管理所有远程数据库服务器连接</p>
      </div>
      <Button variant="primary" size="sm" class="h-[32px] text-[13px]" @click="handleAdd">
        <Plus class="h-3.5 w-3.5 mr-1.5" />添加远程服务器
      </Button>
    </div>

    <!-- Server Cards Grid -->
    <div v-if="serverList.length > 0" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 grid-gap">
      <div
        v-for="row in serverList"
        :key="row.id"
        class="content-card-interactive hover-lift group p-5 flex flex-col gap-3"
      >
        <!-- Name + Type Badge -->
        <div class="flex items-center justify-between">
          <span class="text-[14px] font-semibold truncate" style="color: var(--text-primary)">{{ row.name }}</span>
          <span class="pill pill-active text-[10px] shrink-0 ml-2">{{ row.type === 'mysql' ? 'MySQL' : row.type === 'redis' ? 'Redis' : row.type || 'MySQL' }}</span>
        </div>

        <!-- Host:Port -->
        <div class="font-mono-data text-[13px]" style="color: var(--text-secondary)">{{ row.host }}:{{ row.port }}</div>

        <!-- Username -->
        <div class="text-[12px]" style="color: var(--text-tertiary)">
          <span>用户: </span><span style="color: var(--text-secondary)">{{ row.username }}</span>
        </div>

        <!-- Description -->
        <div v-if="row.description" class="text-[12px] truncate" style="color: var(--text-tertiary)">{{ row.description }}</div>

        <!-- Action Buttons (visible on hover) -->
        <div class="flex items-center gap-2 mt-auto pt-2 opacity-0 group-hover:opacity-100 transition-opacity duration-200" style="border-top: 1px solid var(--border-subtle)">
          <Button variant="ghost" size="xs" class="btn-ghost" @click="handleTest(row)">测试</Button>
          <Button variant="ghost" size="xs" class="btn-ghost" @click="handleEdit(row)">编辑</Button>
          <Button variant="ghost" size="xs" class="btn-ghost-danger" @click="openDeleteDialog(row)">删除</Button>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else class="empty-state mt-16">
      <div class="empty-state-icon">
        <Inbox class="h-12 w-12" />
      </div>
      <div class="empty-state-text">暂无远程服务器</div>
      <Button variant="primary" size="sm" class="h-[32px] text-[13px] mt-4" @click="handleAdd">
        <Plus class="h-3.5 w-3.5 mr-1.5" />添加远程服务器
      </Button>
    </div>

    <RemoteServerDialog v-model="dialogVisible" :type="dialogType" :data="currentServer" @success="loadServers" />

    <Dialog v-model:open="showDeleteDialog">
      <DialogContent class="sm:max-w-[425px] rounded-xl">
        <div class="flex flex-col gap-y-1.5 text-center sm:text-left">
          <DialogTitle class="text-[15px]">删除确认</DialogTitle>
          <DialogDescription class="text-[13px]" style="color: var(--text-tertiary)">
            确定要删除服务器 "{{ deleteTarget?.name }}" 吗？此操作不可恢复。
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
import { ref, onMounted, onActivated } from 'vue'
import { useMessage } from '../composables/useMessage'
import { Plus, Inbox } from 'lucide-vue-next'
import RemoteServerDialog from './RemoteServerDialog.vue'
import { Button } from '@/components/ui/Button.vue'
import { Badge } from '@/components/ui/Badge.vue'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/Table.vue'
import { Dialog, DialogContent, DialogTitle, DialogDescription } from '@/components/ui/Dialog.vue'
import { useHealthStore } from '../stores/health'

const { success, error, warning } = useMessage()
const healthStore = useHealthStore()

const serverList = ref([])
const previousIds = ref(new Set())
const dialogVisible = ref(false)
const dialogType = ref('create')
const currentServer = ref(null)
const showDeleteDialog = ref(false)
const deleteTarget = ref(null)

const loadServers = () => {
  fetch('/api/remote-servers')
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        const newList = data.data || []
        const newIds = new Set(newList.map(s => 'r:' + s.id))
        const newServerIds = [...newIds].filter(id => !previousIds.value.has(id))
        serverList.value = newList
        previousIds.value = newIds
        healthStore.cleanup([...newIds])
        newServerIds.forEach(id => {
          healthStore.forceCheckOne(id)
        })
      }
    })
    .catch((e) => { console.error(e) })
}

const handleAdd = () => { dialogType.value = 'create'; currentServer.value = null; dialogVisible.value = true }
const handleEdit = (row) => { dialogType.value = 'edit'; currentServer.value = { ...row }; dialogVisible.value = true }

const handleTest = (row) => {
  if (row.password === '••••••••') {
    warning('密码为掩码，请先编辑修改密码后再测试连接')
    return
  }
  fetch('/api/remote-servers/test', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ host: row.host, port: row.port, username: row.username, password: row.password }) })
    .then(res => res.json())
    .then(data => { if (data.code === 0) { success('连接成功') } else { error(data.msg || '连接失败') } })
    .catch(() => { error('连接失败') })
}

const openDeleteDialog = (row) => { deleteTarget.value = row; showDeleteDialog.value = true }

const confirmDelete = () => {
  if (!deleteTarget.value) return
  fetch(`/api/remote-servers/${deleteTarget.value.id}`, { method: 'DELETE' })
    .then(res => res.json())
    .then(data => { if (data.code === 0) { success('删除成功'); loadServers() } else { error(data.msg || '删除失败') } })
    .catch(() => { error('删除失败') })
    .finally(() => { showDeleteDialog.value = false })
}

onMounted(() => { loadServers() })
onActivated(() => { loadServers() })
</script>
