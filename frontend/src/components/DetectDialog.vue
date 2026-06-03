<template>
  <Dialog :open="modelValue" @update:open="$emit('update:modelValue', $event)">
    <DialogContent class="max-w-[800px] max-h-[85vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>智能检测数据库</DialogTitle>
        <DialogDescription>应用启动时已自动扫描。Docker 容器会自动提取环境变量获取密码，宿主机端口会尝试常见密码。</DialogDescription>
      </DialogHeader>

      <div class="flex flex-col gap-3.5">
        <div class="flex items-center gap-3">
          <button class="btn-primary" :disabled="scanning" @click="startScan">
            <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': scanning }" />
            {{ scanning ? '扫描中...' : '重新扫描' }}
          </button>
          <span class="text-[11px]" style="color: var(--text-tertiary)">{{ statusText }}</span>
        </div>

        <div v-if="results.length > 0" class="flex flex-col gap-3">
          <div class="flex items-center justify-between">
            <h3 class="text-[13px] font-semibold" style="color: var(--text-primary)">检测到 {{ results.length }} 个实例</h3>
            <div class="flex gap-1.5">
              <span
                class="badge-status text-[10px]"
                :class="tagCounts('mysql').total > 0 ? 'badge-status-info' : 'badge-status-neutral'"
              >
                MySQL {{ tagCounts('mysql').total }}
              </span>
              <span
                class="badge-status badge-status-warning text-[10px]"
              >
                Redis {{ tagCounts('redis').total }}
              </span>
              <span class="badge-status badge-status-success text-[10px]">
                已认证 {{ tagCounts('').authCount }}
              </span>
            </div>
          </div>

          <div class="flex flex-col gap-2">
            <div
              v-for="(row, idx) in results"
              :key="idx"
              class="rounded-xl p-3.5 transition-all duration-200"
              style="background: var(--surface); border: 1px solid var(--border-subtle)"
            >
              <div class="flex items-center gap-2.5">
                <span
                  class="inline-flex items-center justify-center px-2 py-0.5 rounded-md text-[11px] font-semibold"
                  :style="row.type === 'mysql'
                    ? { background: 'var(--accent-soft)', color: 'var(--accent)' }
                    : { background: 'var(--warning-soft)', color: 'var(--warning)' }"
                >
                  {{ row.type === 'mysql' ? 'MySQL' : 'Redis' }}
                </span>
                <span
                  class="badge-status text-[10px]"
                  :class="row.source === 'Docker' ? 'badge-status-warning' : 'badge-status-success'"
                >
                  {{ row.source }}
                </span>
                <span class="font-mono-data text-[13px]" style="color: var(--text-primary)">{{ row.host }}:{{ row.port }}</span>
                <span class="text-[11px] flex-1" style="color: var(--text-tertiary)">{{ row.version }}</span>
                <span
                  class="badge-status text-[10px]"
                  :class="row.status === '已认证' ? 'badge-status-success' : 'badge-status-warning'"
                >
                  {{ row.status }}
                </span>
                <span
                  v-if="row.weakPassword"
                  class="badge-status badge-status-error text-[10px]"
                >
                  <AlertTriangle class="h-3 w-3 mr-0.5" />
                  弱密码
                </span>
                <button
                  v-if="row.status === '已认证'"
                  class="btn-primary text-[11px] h-7 px-2.5"
                  @click="onAdd(row)"
                >
                  一键添加
                </button>
              </div>
              <div v-if="row.status !== '已认证'" class="flex items-center gap-2 mt-2.5 pt-2.5" style="border-top: 1px solid var(--border-subtle)">
                <label class="text-[11px] whitespace-nowrap" style="color: var(--text-tertiary)">用户名</label>
                <Input v-model="row._username" placeholder="root" class="w-[120px] h-7 text-[11px]" />
                <label class="text-[11px] whitespace-nowrap" style="color: var(--text-tertiary)">密码</label>
                <Input v-model="row._password" type="password" placeholder="输入密码" class="w-[150px] h-7 text-[11px]" @keyup.enter="onTestAndAdd(row)" />
                <button class="btn-primary text-[11px] h-7 px-2.5" :disabled="row._testing" @click="onTestAndAdd(row)">
                  <Loader2 v-if="row._testing" class="h-3 w-3 animate-spin" />
                  测试并添加
                </button>
                <span
                  v-if="row._testResult"
                  class="badge-status text-[10px]"
                  :class="row._testResult.type === 'success' ? 'badge-status-success' : 'badge-status-error'"
                >
                  {{ row._testResult.msg }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <div v-if="!scanning && results.length === 0" class="empty-state py-10">
          <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round" class="empty-state-icon"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
          <span class="empty-state-text">未检测到数据库实例</span>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref, computed, reactive, watch } from 'vue'
import { toast } from 'vue-sonner'
import { RefreshCw, Loader2, AlertTriangle } from 'lucide-vue-next'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/Dialog.vue'
import { Button } from '@/components/ui/Button.vue'
import { Input } from '@/components/ui/Input.vue'
import { Badge } from '@/components/ui/Badge.vue'

const emit = defineEmits(['update:modelValue', 'success'])

const props = defineProps({
  modelValue: Boolean
})

const scanning = ref(false)
const results = ref([])

const statusText = computed(() => {
  if (scanning.value) return '正在扫描端口和 Docker 容器...'
  if (results.value.length > 0) return `发现 ${results.value.length} 个实例`
  return '扫描完成，未发现实例'
})

const tagCounts = (type) => {
  const items = results.value
  const mysql = items.filter(i => i.type === 'mysql').length
  const redis = items.filter(i => i.type === 'redis').length
  const auth = items.filter(i => i.status === '已认证').length
  if (type === 'mysql') return { total: mysql }
  if (type === 'redis') return { total: redis }
  return { authCount: auth }
}

const initRow = (item) => reactive({
  ...item,
  _username: item.username || 'root',
  _password: '',
  _testing: false,
  _testResult: null
})

const fetchCached = () => {
  fetch('/api/databases/detect')
    .then(r => r.json())
    .then(data => {
      if (data.code === 0) {
        results.value = (data.data || []).map(initRow)
      }
    })
    .catch((e) => { console.error(e) })
}

const onDialogOpen = () => {
  if (results.value.length === 0) {
    fetchCached()
  }
}

watch(() => props.modelValue, (val) => {
  if (val) onDialogOpen()
})

const startScan = () => {
  scanning.value = true
  fetch('/api/databases/detect', { method: 'POST' })
    .then(r => r.json())
    .then(data => {
      scanning.value = false
      if (data.code === 0) {
        results.value = (data.data || []).map(initRow)
      }
    })
    .catch(() => {
      scanning.value = false
      toast.error('检测请求失败')
    })
}

const onTestAndAdd = async (row) => {
  row._testing = true
  row._testResult = null

  try {
    const testRes = await fetch('/api/databases/db', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: row.name,
        type: row.type,
        host: row.host,
        port: row.port,
        username: row._username || 'root',
        password: row._password || '',
        database: row.type === 'mysql' ? 'mysql' : '',
        version: '',
        description: '智能检测添加',
        permission: '%',
        container: row.container || '',
        testOnly: true
      })
    })
    const testData = await testRes.json()
    if (testData.code === 0) {
      row._testResult = { type: 'success', msg: '连接成功，正在添加...' }

      const addRes = await fetch('/api/databases/db', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: row.name,
          type: row.type,
          host: row.host,
          port: row.port,
          username: row._username || 'root',
          password: row._password || '',
          database: row.type === 'mysql' ? 'mysql' : '',
          version: testData.version || '',
          description: '智能检测添加',
          permission: '%',
          container: row.container || ''
        })
      })
      const addData = await addRes.json()
      if (addData.code === 0) {
        toast.success('添加成功')
        row.status = '已认证'
        row.version = testData.version || ''
        emit('success')
      } else {
        row._testResult = { type: 'error', msg: addData.msg || '添加失败' }
      }
    } else {
      row._testResult = { type: 'error', msg: testData.msg || '连接失败，请检查密码' }
    }
  } catch {
    row._testResult = { type: 'error', msg: '请求失败' }
  }
  row._testing = false
}

const onAdd = (row) => {
  addInstance(row.name, row.type, row.host, row.port, row.username, row.password, row.container)
}

const addInstance = async (name, type, host, port, username, password, container) => {
  try {
    const body = {
      name,
      type,
      host,
      port,
      username: username || 'root',
      password: password || '',
      database: type === 'mysql' ? 'mysql' : '',
      version: '',
      description: '智能检测添加',
      permission: '%',
      container: container || ''
    }

    const res = await fetch('/api/databases/db', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    })
    const data = await res.json()
    if (data.code === 0) {
      toast.success('添加成功')
      emit('success')
    } else {
      toast.error(data.msg || '添加失败')
    }
  } catch {
    toast.error('请求失败')
  }
}
</script>
