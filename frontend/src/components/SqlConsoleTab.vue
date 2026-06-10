<script setup>
// SQL 控制台子组件
// 接收父组件的上下文（serverId、isSQLite、selectedDatabase、apiPrefix、isRemote），
// 内部维护 SQL 文本、执行状态与结果。

import { ref } from 'vue'
import { Textarea } from '@/components/ui/Textarea.vue'
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from '@/components/ui/Table.vue'
import { useMessage } from '../composables/useMessage'
import { useConfirm } from '../composables/useConfirm'

const props = defineProps({
  serverId: { type: [Number, String], default: 0 },
  isSQLite: { type: Boolean, default: false },
  selectedDatabase: { type: String, default: '' },
  apiPrefix: { type: String, required: true },
  isRemote: { type: Boolean, default: false }
})

const toast = useMessage()
const { showConfirm } = useConfirm()

const sqlQuery = ref('')
const sqlResult = ref(null)
const sqlResultLoading = ref(false)

// 后端会做严格白名单校验，前端仅做体验层确认
const DANGEROUS_SQL_PATTERNS = [
  /^\s*DELETE\s+FROM\s+\S+\s*$/i,
  /^\s*DELETE\s+FROM\s+\S+\s*;\s*$/i,
  /^\s*UPDATE\s+\S+\s+SET\s+.+\s*$/i,
  /^\s*TRUNCATE\s+/i,
  /^\s*DROP\s+/i,
]

const buildExecuteBody = (sql) => {
  const body = { server_id: props.serverId, sql }
  if (!props.isSQLite) body.database = props.selectedDatabase
  return body
}

const sourceParam = (isRemote) => isRemote ? 'source=remote' : 'source=local'

const executeSql = async () => {
  if (!sqlQuery.value.trim()) { toast.warning('请输入SQL语句'); return }
  if (!props.serverId) { toast.warning('请先选择一个实例'); return }
  const sqlTrimmed = sqlQuery.value.trim()
  const isDangerous = DANGEROUS_SQL_PATTERNS.some(p => p.test(sqlTrimmed))
  if (isDangerous) {
    const ok = await showConfirm({
      title: '请确认',
      text: '该SQL语句可能造成数据不可逆的修改，确认执行？',
      variant: 'warning',
      confirmText: '确认执行'
    })
    if (!ok) return
  }
  sqlResultLoading.value = true
  sqlResult.value = null
  fetch(`${props.apiPrefix}/execute?${sourceParam(props.isRemote)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(buildExecuteBody(sqlQuery.value))
  })
    .then(res => res.json()).then(data => {
      if (data.code === 0) {
        if (data.data.rows && data.data.rows.length > 0) {
          sqlResult.value = { type: 'rows', data: data.data.rows, columns: Object.keys(data.data.rows[0]) }
          toast.success(`查询成功，返回 ${data.data.rows.length} 行`)
        } else {
          sqlResult.value = { type: 'message', data: data.data.msg || '执行成功' }
          toast.success(data.data.msg || '执行成功')
        }
      } else {
        sqlResult.value = { type: 'error', data: data.msg }
        toast.error(data.msg)
      }
    })
    .catch(err => {
      sqlResult.value = { type: 'error', data: err.message }
      toast.error('执行失败: ' + err.message)
    })
    .finally(() => { sqlResultLoading.value = false })
}

defineExpose({ sqlQuery, sqlResult, sqlResultLoading })
</script>

<template>
  <div class="sql-console">
    <Textarea v-model="sqlQuery" :rows="5" placeholder="输入SQL查询语句..." class="font-mono text-sm mb-3 code-editor" style="border-color: var(--border)" />
    <div class="flex justify-end mb-3">
      <button class="btn-primary" @click="executeSql" :disabled="sqlResultLoading">
        {{ sqlResultLoading ? '执行中...' : '执行' }}
      </button>
    </div>
    <div v-if="sqlResult" class="mt-2">
      <div v-if="sqlResult.type === 'rows'" class="sql-result-table">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead v-for="col in sqlResult.columns" :key="col" class="text-xs">{{ col }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="(row, idx) in sqlResult.data" :key="idx">
              <TableCell v-for="col in sqlResult.columns" :key="col" class="text-xs font-mono-data">{{ row[col] ?? 'NULL' }}</TableCell>
            </TableRow>
          </TableBody>
        </Table>
        <div class="sql-result-footer" style="color: var(--text-tertiary)">共 {{ sqlResult.data.length }} 行</div>
      </div>
      <div v-else-if="sqlResult.type === 'message'" class="sql-result-msg sql-result-msg--success">{{ sqlResult.data }}</div>
      <div v-else-if="sqlResult.type === 'error'" class="sql-result-msg sql-result-msg--error">{{ sqlResult.data }}</div>
    </div>
  </div>
</template>

<style scoped>
.sql-console { width: 100%; }
.code-editor { font-family: 'Cascadia Code', 'Fira Code', 'Source Code Pro', monospace; }
.sql-result-table { max-width: 100%; overflow: auto; max-height: 60vh; border: 1px solid var(--border); border-radius: 4px; }
.sql-result-footer { padding: 6px 8px; font-size: 12px; }
.sql-result-msg { padding: 10px 12px; border-radius: 4px; font-size: 13px; }
.sql-result-msg--success { background: rgba(34, 197, 94, 0.1); color: #16a34a; }
.sql-result-msg--error { background: rgba(239, 68, 68, 0.1); color: #dc2626; }
.font-mono-data { font-family: 'Cascadia Code', 'Fira Code', 'Source Code Pro', monospace; }
</style>
