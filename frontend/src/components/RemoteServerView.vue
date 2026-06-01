<template>
  <div class="page-padding h-full overflow-y-auto">
    <div class="content-card">
      <div class="content-header">
        <div class="flex items-start justify-between">
          <div>
            <h2 class="text-[15px] font-semibold text-foreground">远程服务器</h2>
            <p class="text-[13px] text-muted-foreground mt-0.5">管理所有远程数据库服务器连接</p>
          </div>
          <Button variant="primary" size="sm" class="h-[32px] text-[13px]" @click="handleAdd">
            <Plus class="h-3.5 w-3.5 mr-1.5" />添加远程服务器
          </Button>
        </div>
      </div>

      <div class="border-t border-border section-padding overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow class="hover:bg-transparent border-b border-border">
              <TableHead class="min-w-[150px] text-[12px] font-normal text-muted-foreground h-10">名称</TableHead>
              <TableHead class="w-[80px] text-[12px] font-normal text-muted-foreground h-10">类型</TableHead>
              <TableHead class="min-w-[180px] text-[12px] font-normal text-muted-foreground h-10">数据库地址</TableHead>
              <TableHead class="w-[64px] text-[12px] font-normal text-muted-foreground h-10">端口</TableHead>
              <TableHead class="w-[100px] text-[12px] font-normal text-muted-foreground h-10">用户名</TableHead>
              <TableHead class="w-[90px] text-[12px] font-normal text-muted-foreground h-10">密码</TableHead>
              <TableHead class="min-w-[150px] text-[12px] font-normal text-muted-foreground h-10">描述信息</TableHead>
              <TableHead class="text-center w-[180px] text-[12px] font-normal text-muted-foreground h-10">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <template v-if="serverList.length === 0">
              <TableRow>
                <TableCell colspan="8" class="h-64 text-center">
                  <div class="flex flex-col items-center justify-center text-muted-foreground">
                    <Inbox class="h-10 w-10 mb-2 opacity-30" />
                    <p class="text-[13px]">暂无远程服务器</p>
                  </div>
                </TableCell>
              </TableRow>
            </template>
            <template v-for="row in serverList" :key="row.id">
              <TableRow class="hover:bg-muted transition-colors duration-150 border-b border-border">
                <TableCell><span class="text-[13px] text-foreground">{{ row.name }}</span></TableCell>
                <TableCell>
                  <Badge :variant="row.type === 'mysql' ? 'default' : 'secondary'" class="rounded-full text-[10px] py-0">
                    {{ row.type === 'mysql' ? 'MySQL' : row.type === 'redis' ? 'Redis' : row.type || 'MySQL' }}
                  </Badge>
                </TableCell>
                <TableCell><span class="text-[13px] text-secondary-foreground">{{ row.host }}</span></TableCell>
                <TableCell><span class="text-[13px] text-muted-foreground">{{ row.port }}</span></TableCell>
                <TableCell><span class="text-[13px] text-muted-foreground">{{ row.username }}</span></TableCell>
                <TableCell><span class="text-[13px] text-muted-foreground">••••••••</span></TableCell>
                <TableCell><span class="text-[13px] text-muted-foreground">{{ row.description || '-' }}</span></TableCell>
                <TableCell>
                  <div class="flex items-center gap-0.5">
                    <Button variant="ghost" size="xs" class="text-primary hover:bg-primary/10" @click="handleTest(row)">连接测试</Button>
                    <Button variant="ghost" size="xs" class="text-primary hover:bg-primary/10" @click="handleEdit(row)">编辑</Button>
                    <Button variant="ghost" size="xs" class="text-destructive hover:bg-destructive/10" @click="openDeleteDialog(row)">删除</Button>
                  </div>
                </TableCell>
              </TableRow>
            </template>
          </TableBody>
        </Table>
      </div>
    </div>

    <RemoteServerDialog v-model="dialogVisible" :type="dialogType" :data="currentServer" @success="loadServers" />

    <Dialog v-model:open="showDeleteDialog">
      <DialogContent class="sm:max-w-[425px] rounded-xl">
        <div class="flex flex-col gap-y-1.5 text-center sm:text-left">
          <DialogTitle class="text-[15px]">删除确认</DialogTitle>
          <DialogDescription class="text-[13px] text-muted-foreground">
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
