<template>
  <Dialog :open="modelValue" @update:open="$emit('update:modelValue', $event)">
    <DialogContent class="max-w-[680px] max-h-[85vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>智能检测数据库</DialogTitle>
        <DialogDescription>自动扫描 Docker 容器和本地端口</DialogDescription>
      </DialogHeader>

      <div class="flex flex-col gap-4">
        <!-- Scan button + status -->
        <div class="flex items-center justify-between">
          <button class="btn-primary" :disabled="scanning" @click="startScan">
            <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': scanning }" />
            {{ scanning ? '扫描中...' : '重新扫描' }}
          </button>
          <span class="text-[12px]" style="color: var(--text-tertiary)">{{ statusText }}</span>
        </div>

        <!-- Filter pills -->
        <div v-if="results.length > 0" class="flex items-center gap-1.5 flex-wrap">
          <span
            :class="['pill', detectFilter === 'all' ? 'pill-active' : 'pill-default']"
            @click="detectFilter = 'all'"
          >全部 {{ results.length }}</span>
          <span
            v-for="t in uniqueTypes"
            :key="t"
            :class="['pill', detectFilter === t ? 'pill-active' : 'pill-default']"
            @click="detectFilter = detectFilter === t ? 'all' : t"
          >{{ getTypeLabel(t) }} {{ countByType(t) }}</span>
          <span
            :class="['pill', detectFilter === 'auth' ? 'pill-active' : 'pill-default']"
            @click="detectFilter = detectFilter === 'auth' ? 'all' : 'auth'"
          >已认证 {{ countAuth }}</span>
        </div>

        <!-- Instance cards -->
        <div v-if="filteredResults.length > 0" class="flex flex-col gap-2.5">
          <div
            v-for="(row, idx) in filteredResults"
            :key="idx"
            class="rounded-xl overflow-hidden transition-all duration-200"
            :style="row.ignored
              ? { background: 'var(--surface-muted)', border: '1px dashed var(--border-subtle)', opacity: 0.55 }
              : { background: 'var(--surface)', border: '1px solid var(--border-subtle)' }"
          >
            <!-- Main row -->
            <div class="flex items-center gap-3 px-4 py-3">
              <!-- Type icon circle -->
              <div
                class="w-9 h-9 rounded-lg flex items-center justify-center text-[13px] font-bold shrink-0"
                :style="{ background: getTypeSoftColor(row.type), color: getTypeColor(row.type) }"
              >
                {{ getTypeLabel(row.type).charAt(0) }}
              </div>

              <!-- Info block -->
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-1.5">
                  <span class="text-[13px] font-semibold truncate" style="color: var(--text-primary)">
                    {{ row.source === 'Docker' ? row.container : getTypeLabel(row.type) }}
                  </span>
                  <span
                    v-if="row.source === 'Docker'"
                    class="text-[10px] px-1.5 py-0.5 rounded font-medium shrink-0"
                    style="background: rgba(245,158,11,0.1); color: #d97706"
                  >Docker</span>
                  <span
                    v-if="row.version"
                    class="text-[10px] px-1.5 py-0.5 rounded font-medium shrink-0"
                    :style="{ background: getTypeSoftColor(row.type), color: getTypeColor(row.type) }"
                  >{{ getTypeLabel(row.type) }}:{{ shortVersion(row.version) }}</span>
                  <span v-if="row.weakPassword" class="text-[10px] px-1.5 py-0.5 rounded font-medium shrink-0" style="background: rgba(239,68,68,0.1); color: #dc2626">
                    弱密码
                  </span>
                </div>
                <div class="flex items-center gap-2 mt-0.5">
                  <span class="font-mono-data text-[12px]" style="color: var(--text-secondary)">
                    {{ row.host }}:{{ row.port }}
                  </span>
                  <template v-if="row.source === 'Docker' && row.reachableFrom?.length">
                    <span class="font-mono-data text-[11px]" style="color: var(--text-tertiary)">
                      {{ extractContainerIP(row) }}
                    </span>
                  </template>
                </div>
              </div>

              <!-- Status + Actions -->
              <div class="flex items-center gap-2 shrink-0">
                <span
                  class="text-[11px] font-medium px-2 py-0.5 rounded-full"
                  :style="row.status === '已认证'
                    ? { background: 'rgba(16,185,129,0.1)', color: '#059669' }
                    : { background: 'rgba(245,158,11,0.1)', color: '#d97706' }"
                >{{ row.status }}</span>

                <button
                  v-if="row.status === '已认证' && !row.ignored"
                  class="btn-primary text-[11px] h-7 px-3"
                  @click="onAdd(row)"
                >添加</button>

                <button
                  v-if="row.status !== '已认证' && !row.ignored"
                  class="btn-secondary text-[11px] h-7 px-2.5"
                  @click="toggleExpand(row)"
                >{{ expandedRows.has(idx) ? '收起' : '认证' }}</button>

                <button
                  class="btn-ghost text-[11px] h-7 px-1.5"
                  @click="row.ignored ? onUnignore(row) : onIgnore(row)"
                >
                  <EyeOff v-if="!row.ignored" class="h-3.5 w-3.5" />
                  <Eye v-else class="h-3.5 w-3.5" />
                </button>
              </div>
            </div>

            <!-- Auth form (expandable) -->
            <div
              v-if="row.status !== '已认证' && !row.ignored && expandedRows.has(idx)"
              class="px-4 pb-3 pt-2"
              style="border-top: 1px solid var(--border-subtle)"
            >
              <div class="flex items-center gap-2">
                <div class="flex items-center gap-1.5 flex-1">
                  <Input v-model="row._username" placeholder="用户名" class="h-7 text-[12px] flex-1" />
                  <Input v-model="row._password" type="password" placeholder="密码" class="h-7 text-[12px] flex-1" @keyup.enter="onTestAndAdd(row)" />
                </div>
                <button class="btn-primary text-[11px] h-7 px-3" :disabled="row._testing" @click="onTestAndAdd(row)">
                  <Loader2 v-if="row._testing" class="h-3 w-3 animate-spin" />
                  连接
                </button>
              </div>
              <div v-if="row._testResult" class="mt-1.5 text-[11px]" :style="{ color: row._testResult.type === 'success' ? '#059669' : '#dc2626' }">
                {{ row._testResult.msg }}
              </div>
            </div>
          </div>
        </div>

        <!-- Empty state -->
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
import { RefreshCw, Loader2, EyeOff, Eye } from 'lucide-vue-next'
import { getTypeLabel, getTypeColor, getTypeSoftColor } from '@/lib/utils'
import { detectApi, databaseApi } from '@/api/database'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/Dialog.vue'
import { Input } from '@/components/ui/Input.vue'

const emit = defineEmits(['update:modelValue', 'success'])
const props = defineProps({ modelValue: Boolean })

const scanning = ref(false)
const results = ref([])
const detectFilter = ref('all')
const expandedRows = ref(new Set())

const toggleExpand = (row) => {
  const idx = results.value.indexOf(row)
  if (idx >= 0) {
    const s = new Set(expandedRows.value)
    if (s.has(idx)) s.delete(idx); else s.add(idx)
    expandedRows.value = s
  }
}

// "PostgreSQL 16.14 (Debian ...)" → "16.14"
const shortVersion = (v) => {
  if (!v) return ''
  const m = v.match(/(\d+\.\d+(?:\.\d+)?)/)
  return m ? m[1] : v.split(' ')[0]
}

// 从 reachableFrom 中提取容器内网地址（后端已带"容器内网"后缀）
const extractContainerIP = (row) => {
  if (!row.reachableFrom?.length) return ''
  const internal = row.reachableFrom.find(a => a.includes('容器内网'))
  return internal || ''
}

const uniqueTypes = computed(() => [...new Set(results.value.map(i => i.type))])
const countByType = (type) => results.value.filter(i => i.type === type).length
const countAuth = computed(() => results.value.filter(i => i.status === '已认证').length)

const filteredResults = computed(() => {
  const items = results.value
  if (detectFilter.value === 'all') return items
  if (detectFilter.value === 'auth') return items.filter(i => i.status === '已认证')
  return items.filter(i => i.type === detectFilter.value)
})

const statusText = computed(() => {
  if (scanning.value) return '正在扫描...'
  if (results.value.length > 0) return `发现 ${results.value.length} 个实例`
  return '未发现实例'
})

const initRow = (item) => reactive({
  ...item,
  _username: item.username || (item.type === 'postgresql' ? 'postgres' : 'root'),
  _password: '',
  _testing: false,
  _testResult: null
})

const fetchCached = () => {
  Promise.all([detectApi.list(true), detectApi.ignoredList(), databaseApi.list('all')])
    .then(([res, igRes, dbRes]) => {
      const ignored = new Set((igRes.data?.data || []).map(i => i.fingerprint))
      // 已添加的实例：按 host:port:type 去重
      const existing = new Set()
      const dbList = dbRes.data?.data || []
      dbList.forEach(db => {
        if (db.host && db.port) {
          existing.add(`${db.host}:${db.port}:${db.type}`)
        }
      })
      const items = (res.data?.data || []).map(it => ({
        ...it,
        ignored: ignored.has(it.fingerprint),
        alreadyAdded: existing.has(`${it.host}:${it.port}:${it.type}`)
      })).filter(it => !it.alreadyAdded)
      results.value = items.map(initRow)
    })
    .catch(console.error)
}

watch(() => props.modelValue, (val) => {
  if (val) fetchCached()
})

const startScan = () => {
  scanning.value = true
  detectApi.scan()
    .then(() => { scanning.value = false; fetchCached() })
    .catch(() => { scanning.value = false; toast.error('检测请求失败') })
}

const onIgnore = async (row) => {
  try {
    await detectApi.ignore(row.fingerprint, row.name)
    row.ignored = true
    toast.success(`已忽略`)
  } catch { toast.error('忽略失败') }
}

const onUnignore = async (row) => {
  try {
    await detectApi.unignore(row.fingerprint)
    row.ignored = false
    toast.success(`已恢复`)
  } catch { toast.error('恢复失败') }
}

const onTestAndAdd = async (row) => {
  row._testing = true
  row._testResult = null
  try {
    const testRes = await fetch('/api/databases/db', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: row.name, type: row.type, host: row.host, port: row.port,
        username: row._username || 'root', password: row._password || '',
        database: row.type === 'mysql' ? 'mysql' : '',
        version: '', description: '智能检测添加', permission: '%',
        container: row.container || '', testOnly: true
      })
    })
    const testData = await testRes.json()
    if (testData.code === 0) {
      row._testResult = { type: 'success', msg: '连接成功' }
      const addRes = await fetch('/api/databases/db', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: row.name, type: row.type, host: row.host, port: row.port,
          username: row._username || 'root', password: row._password || '',
          database: row.type === 'mysql' ? 'mysql' : '',
          version: testData.version || '', description: '智能检测添加',
          permission: '%', container: row.container || ''
        })
      })
      const addData = await addRes.json()
      if (addData.code === 0) {
        toast.success('添加成功')
        // 从检测列表中移除已添加的行
        const idx = results.value.indexOf(row)
        if (idx >= 0) results.value.splice(idx, 1)
        emit('success')
      } else {
        row._testResult = { type: 'error', msg: addData.msg || '添加失败' }
      }
    } else {
      row._testResult = { type: 'error', msg: testData.msg || '连接失败' }
    }
  } catch {
    row._testResult = { type: 'error', msg: '请求失败' }
  }
  row._testing = false
}

const onAdd = async (row) => {
  await addInstance(row.name, row.type, row.host, row.port, row.username, row.password, row.container, row)
}

const addInstance = async (name, type, host, port, username, password, container, row = null) => {
  try {
    const res = await fetch('/api/databases/db', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name, type, host, port,
        username: username || 'root', password: password || '',
        database: type === 'mysql' ? 'mysql' : '',
        version: '', description: '智能检测添加',
        permission: '%', container: container || ''
      })
    })
    const data = await res.json()
    if (data.code === 0) {
      toast.success('添加成功')
      // 从检测列表中移除已添加的行
      if (row) {
        const idx = results.value.indexOf(row)
        if (idx >= 0) results.value.splice(idx, 1)
      }
      emit('success')
    } else {
      toast.error(data.msg || '添加失败')
    }
  } catch { toast.error('请求失败') }
}
</script>
