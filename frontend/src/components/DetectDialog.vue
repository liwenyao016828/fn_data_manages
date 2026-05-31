<template>
  <Dialog :open="modelValue" @update:open="$emit('update:modelValue', $event)">
    <DialogContent class="max-w-[800px] max-h-[85vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>智能检测数据库</DialogTitle>
        <DialogDescription>应用启动时已自动扫描。Docker 容器会自动提取环境变量获取密码，宿主机端口会尝试常见密码。</DialogDescription>
      </DialogHeader>

      <div class="flex flex-col gap-3.5">
        <div class="flex items-center gap-3">
          <Button size="sm" :disabled="scanning" @click="startScan">
            <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': scanning }" />
            {{ scanning ? '扫描中...' : '重新扫描' }}
          </Button>
          <span class="text-xs text-muted-foreground">{{ statusText }}</span>
        </div>

        <div v-if="results.length > 0" class="flex flex-col gap-3">
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-semibold text-foreground">检测到 {{ results.length }} 个实例</h3>
            <div class="flex gap-1.5">
              <Badge
                :variant="tagCounts('mysql').total > 0 ? 'default' : 'secondary'"
                class="rounded-full"
              >
                MySQL {{ tagCounts('mysql').total }}
              </Badge>
              <Badge
                :variant="tagCounts('redis').total > 0 ? 'secondary' : 'outline'"
                class="rounded-full bg-amber-500/10 text-amber-600 border-amber-200"
              >
                Redis {{ tagCounts('redis').total }}
              </Badge>
              <Badge variant="secondary" class="rounded-full bg-emerald-500/10 text-emerald-600 border-emerald-200">
                已认证 {{ tagCounts('').authCount }}
              </Badge>
            </div>
          </div>

          <div class="rounded-xl border border-border overflow-hidden">
            <div v-for="(row, idx) in results" :key="idx" class="border-b border-border last:border-b-0">
              <div class="flex items-center gap-2.5 px-3.5 py-2.5 bg-background text-sm">
                <span
                  class="inline-block px-2 py-0.5 rounded text-xs font-semibold text-white"
                  :class="row.type === 'mysql' ? 'bg-blue-400' : 'bg-amber-500'"
                >
                  {{ row.type === 'mysql' ? 'MySQL' : 'Redis' }}
                </span>
                <Badge
                  :variant="row.source === 'Docker' ? 'outline' : 'secondary'"
                  :class="row.source === 'Docker' ? 'bg-amber-500/10 text-amber-600 border-amber-200' : 'bg-emerald-500/10 text-emerald-600 border-emerald-200'"
                  class="text-xs"
                >
                  {{ row.source }}
                </Badge>
                <span class="font-mono text-sm text-foreground">{{ row.host }}:{{ row.port }}</span>
                <span class="text-xs text-muted-foreground flex-1">{{ row.version }}</span>
                <Badge
                  :variant="row.status === '已认证' ? 'secondary' : 'outline'"
                  :class="row.status === '已认证' ? 'bg-emerald-500/10 text-emerald-600 border-emerald-200' : 'bg-amber-500/10 text-amber-600 border-amber-200'"
                  class="text-xs"
                >
                  {{ row.status }}
                </Badge>
                <Badge
                  v-if="row.weakPassword"
                  variant="outline"
                  class="bg-red-500/10 text-red-600 border-red-200 text-xs"
                >
                  <AlertTriangle class="h-3 w-3 mr-0.5" />
                  弱密码
                </Badge>
                <Button
                  v-if="row.status === '已认证'"
                  size="sm"
                  @click="onAdd(row)"
                >
                  一键添加
                </Button>
              </div>
              <div v-if="row.status !== '已认证'" class="flex items-center gap-2 px-3.5 py-2.5 bg-muted/50 border-t border-dashed border-border text-sm">
                <label class="text-xs text-muted-foreground whitespace-nowrap">用户名</label>
                <Input v-model="row._username" placeholder="root" class="w-[120px] h-8 text-xs" />
                <label class="text-xs text-muted-foreground whitespace-nowrap">密码</label>
                <Input v-model="row._password" type="password" placeholder="输入密码" class="w-[150px] h-8 text-xs" @keyup.enter="onTestAndAdd(row)" />
                <Button size="sm" :disabled="row._testing" @click="onTestAndAdd(row)">
                  <Loader2 v-if="row._testing" class="h-3 w-3 animate-spin" />
                  测试并添加
                </Button>
                <div
                  v-if="row._testResult"
                  class="px-2.5 py-1 rounded text-xs whitespace-nowrap"
                  :class="row._testResult.type === 'success'
                    ? 'bg-emerald-500/10 text-emerald-600 border border-emerald-200'
                    : 'bg-destructive/10 text-destructive border border-destructive/20'"
                >
                  {{ row._testResult.msg }}
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-if="!scanning && results.length === 0" class="flex flex-col items-center justify-center py-10 text-muted-foreground">
          <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round" class="mb-3 opacity-30"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
          <span class="text-sm">未检测到数据库实例</span>
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
