<template>
  <Dialog :open="modelValue" @update:open="handleOpenChange">
    <DialogContent class="max-w-[480px]">
      <DialogHeader>
        <DialogTitle>{{ type === 'create' ? '添加连接' : '编辑连接' }}</DialogTitle>
      </DialogHeader>

      <div class="flex flex-col gap-4">
        <div class="grid grid-cols-2 gap-4">
          <div class="flex flex-col gap-1.5">
            <label class="text-[12px] font-medium" style="color: var(--text-secondary)">名称 <span style="color: var(--danger)">*</span></label>
            <Input v-model="form.name" placeholder="实例名称" />
          </div>
          <div class="flex flex-col gap-1.5">
            <label class="text-[12px] font-medium" style="color: var(--text-secondary)">类型 <span style="color: var(--danger)">*</span></label>
            <div class="flex gap-1 flex-wrap">
              <button
                type="button"
                class="inline-flex items-center justify-center px-3 h-8 text-[12px] font-medium rounded-lg transition-all duration-200 cursor-pointer"
                :style="form.type === 'mysql' ? { background: 'var(--accent-soft)', color: 'var(--accent)', border: '1px solid var(--accent)' } : { background: 'var(--muted)', color: 'var(--text-tertiary)', border: '1px solid transparent' }"
                @click="form.type = 'mysql'"
              >
                MySQL
              </button>
              <button
                type="button"
                class="inline-flex items-center justify-center px-3 h-8 text-[12px] font-medium rounded-lg transition-all duration-200 cursor-pointer"
                :style="form.type === 'mariadb' ? { background: 'var(--accent-soft)', color: 'var(--accent)', border: '1px solid var(--accent)' } : { background: 'var(--muted)', color: 'var(--text-tertiary)', border: '1px solid transparent' }"
                @click="form.type = 'mariadb'"
              >
                MariaDB
              </button>
              <button
                type="button"
                class="inline-flex items-center justify-center px-3 h-8 text-[12px] font-medium rounded-lg transition-all duration-200 cursor-pointer"
                :style="form.type === 'postgresql' ? { background: 'rgba(99, 102, 241, 0.1)', color: '#6366f1', border: '1px solid #6366f1' } : { background: 'var(--muted)', color: 'var(--text-tertiary)', border: '1px solid transparent' }"
                @click="form.type = 'postgresql'"
              >
                PostgreSQL
              </button>
              <button
                type="button"
                class="inline-flex items-center justify-center px-3 h-8 text-[12px] font-medium rounded-lg transition-all duration-200 cursor-pointer"
                :style="form.type === 'redis' ? { background: 'var(--warning-soft)', color: 'var(--warning)', border: '1px solid var(--warning)' } : { background: 'var(--muted)', color: 'var(--text-tertiary)', border: '1px solid transparent' }"
                @click="form.type = 'redis'"
              >
                Redis
              </button>
              <button
                type="button"
                class="inline-flex items-center justify-center px-3 h-8 text-[12px] font-medium rounded-lg transition-all duration-200 cursor-pointer"
                :style="form.type === 'sqlite' ? { background: 'var(--success-soft)', color: 'var(--success)', border: '1px solid var(--success)' } : { background: 'var(--muted)', color: 'var(--text-tertiary)', border: '1px solid transparent' }"
                @click="form.type = 'sqlite'"
              >
                SQLite
              </button>
            </div>
          </div>
        </div>

        <div class="grid grid-cols-3 gap-4">
          <div :class="form.type === 'sqlite' ? 'col-span-3' : 'col-span-2'" class="flex flex-col gap-1.5">
            <label class="text-[12px] font-medium" style="color: var(--text-secondary)">{{ form.type === 'sqlite' ? '文件路径' : '主机地址' }} <span style="color: var(--danger)">*</span></label>
            <Input v-model="form.host" :placeholder="form.type === 'sqlite' ? '/path/to/database.db' : 'localhost'" />
          </div>
          <div v-if="form.type !== 'sqlite'" class="flex flex-col gap-1.5">
            <label class="text-[12px] font-medium" style="color: var(--text-secondary)">端口 <span style="color: var(--danger)">*</span></label>
            <Input v-model="form.port" type="number" :min="1" :max="65535" />
          </div>
        </div>

        <div v-if="form.type !== 'sqlite'" class="grid grid-cols-2 gap-4">
          <div class="flex flex-col gap-1.5">
            <label class="text-[12px] font-medium" style="color: var(--text-secondary)">用户名 <span style="color: var(--danger)">*</span></label>
            <Input v-model="form.username" placeholder="用户名" />
          </div>
          <div class="flex flex-col gap-1.5">
            <label class="text-[12px] font-medium" style="color: var(--text-secondary)">密码</label>
            <Input v-model="form.password" type="password" placeholder="密码" />
          </div>
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-[12px] font-medium" style="color: var(--text-secondary)">数据库名</label>
          <Input v-model="form.database" placeholder="默认与实例名相同" />
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-[12px] font-medium" style="color: var(--text-secondary)">描述</label>
          <Textarea v-model="form.description" :rows="2" placeholder="可选" />
        </div>
      </div>

      <div v-if="duplicateInstance" class="mt-3 p-3 rounded-xl" style="background: var(--warning-soft); border: 1px solid color-mix(in srgb, var(--warning) 20%, transparent)">
        <div class="flex items-start gap-2">
          <AlertTriangle class="h-4 w-4 shrink-0 mt-0.5" style="color: var(--warning)" />
          <div class="flex-1 min-w-0">
            <p class="text-[12px] font-medium" style="color: var(--warning)">连接已存在</p>
            <p class="text-[11px] mt-0.5" style="color: var(--warning)">
              该主机({{ form.host }})、用户名({{ form.username }})和端口({{ form.port }})的数据库连接已存在
            </p>
            <div class="flex items-center gap-2 mt-2">
              <span class="text-[11px]" style="color: var(--warning)">{{ duplicateInstance.name }}</span>
              <Badge v-if="duplicateInstance.isRemote" variant="outline" class="badge-status badge-status-warning text-[10px] h-[16px]">远程</Badge>
              <span class="text-[11px] font-mono-data" style="color: var(--warning)">{{ getTypeLabel(duplicateInstance.type) }}</span>
            </div>
            <button class="btn-ghost mt-1 text-[11px]" style="color: var(--warning)" @click="goToDuplicate">
              跳转到该连接 →
            </button>
          </div>
        </div>
      </div>

      <div class="flex justify-between items-center mt-4">
        <div class="flex items-center gap-2">
          <button class="btn-secondary" @click="testConnection" :disabled="testLoading">
            <Loader2 v-if="testLoading" class="h-4 w-4 animate-spin" />
            <Zap v-else class="h-4 w-4" />
            测试连接
          </button>
          <span v-if="testResult !== null" class="text-[11px] font-medium" :style="{ color: testResult ? 'var(--success)' : 'var(--danger)' }">
            {{ testResult ? '连接成功' : '连接失败' }}
          </span>
        </div>
        <div class="flex gap-2">
          <button class="btn-ghost" @click="handleClose">取消</button>
          <button class="btn-primary" @click="handleSubmit" :disabled="loading || !!duplicateInstance">
            <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
            确认
          </button>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { reactive, watch, ref, computed } from 'vue'
import { toast } from 'vue-sonner'
import { Loader2, Zap, AlertTriangle } from 'lucide-vue-next'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/Dialog.vue'
import { Button } from '@/components/ui/Button.vue'
import { Input } from '@/components/ui/Input.vue'
import { Textarea } from '@/components/ui/Textarea.vue'
import { Badge } from '@/components/ui/Badge.vue'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/Select.vue'
import { databaseApi } from '../api/database'
import { getTypeLabel } from '@/lib/utils'

const props = defineProps({
  modelValue: Boolean,
  type: String,
  data: Object,
  existingInstances: { type: Array, default: () => [] }
})

const emit = defineEmits(['update:modelValue', 'close', 'success', 'navigateToInstance'])

const loading = ref(false)
const testLoading = ref(false)
const testResult = ref(null)

const form = reactive({
  id: 0,
  name: '',
  type: 'mysql',
  host: 'localhost',
  port: 3306,
  username: '',
  password: '',
  version: '',
  description: '',
  database: '',
  ssl: false,
  permission: '%',
  isRemote: false
})

const getDefaultPort = (type) => {
  if (type === 'redis') return 6379
  if (type === 'postgresql') return 5432
  if (type === 'sqlite') return 0
  return 3306 // mysql, mariadb
}

watch(() => form.type, (newType) => {
  form.port = getDefaultPort(newType)
  if (newType === 'sqlite') {
    form.host = ''
    form.username = ''
    form.password = ''
  } else if (!form.host || form.host === '') {
    form.host = 'localhost'
  }
})

const duplicateInstance = computed(() => {
  if (props.type === 'edit') return null
  if (!form.host || !form.username || !form.port) return null

  const formHost = String(form.host || 'localhost').trim()
  const formUsername = String(form.username || '').trim()
  const formPort = Number(form.port || 3306)

  return props.existingInstances.find(inst => {
    const instHost = String(inst.host || 'localhost').trim()
    const instUsername = String(inst.username || '').trim()
    const instPort = Number(inst.port || 3306)

    return instHost === formHost
      && instUsername === formUsername
      && instPort === formPort
  }) || null
})

const goToDuplicate = () => {
  if (duplicateInstance.value) {
    emit('navigateToInstance', duplicateInstance.value)
    emit('update:modelValue', false)
  }
}

const handleOpenChange = (val) => {
  emit('update:modelValue', val)
  if (!val) emit('close')
}

watch(() => props.data, (val) => {
  if (val) {
    form.id = val.id || 0
    form.name = val.name || ''
    form.type = val.type || 'mysql'
    form.host = val.host || 'localhost'
    form.port = val.port || getDefaultPort(val.type)
    form.username = val.username || ''
    form.password = val.password || ''
    form.version = val.version || ''
    form.description = val.description || ''
    form.database = val.database || ''
    form.ssl = val.ssl || false
    form.permission = val.permission || '%'
    form.isRemote = val.isRemote || false
  }
}, { immediate: true })

watch(() => props.modelValue, (val) => {
  if (val) {
    testResult.value = null
    if (props.type === 'create') {
    form.id = 0
    form.name = ''
    form.type = 'mysql'
    form.host = 'localhost'
    form.port = 3306
    form.username = ''
    form.password = ''
    form.version = ''
    form.description = ''
    form.database = ''
    form.ssl = false
    form.permission = '%'
    form.isRemote = false
    }
  }
})

const testConnection = () => {
  if (!form.host) { toast.warning(form.type === 'sqlite' ? '请输入文件路径' : '请输入主机地址'); return }
  if (form.type !== 'sqlite' && !form.port) { toast.warning('请输入端口'); return }
  if (form.type !== 'sqlite' && !form.username) { toast.warning('请输入用户名'); return }

  testLoading.value = true
  testResult.value = null
  const passwordChanged = form.password !== '••••••••'
  const payload = {
    name: form.name || '__test__',
    type: form.type,
    host: form.host,
    port: parseInt(form.port) || 3306,
    username: form.username,
    password: passwordChanged ? form.password : undefined,
    database: form.database || undefined,
    version: form.version,
    description: '',
    ssl: form.ssl,
    permission: '%',
    testOnly: true
  }
  if (props.type === 'edit' && form.id) {
    payload.testId = form.id
    payload.testSource = form.isRemote ? 'remote' : 'local'
  }
  databaseApi.create(payload).then(res => {
    testResult.value = res.data.code === 0
    if (res.data.code === 0) {
      toast.success('连接成功' + (res.data.version ? ` (${res.data.version})` : ''))
    } else {
      const detail = res.data.msg || '连接失败'
      toast.error(detail)
    }
  }).catch(err => {
    testResult.value = false
    toast.error('连接失败: ' + (err.message || '网络错误'))
  }).finally(() => {
    testLoading.value = false
  })
}

const handleClose = () => {
  emit('update:modelValue', false)
}

const handleSubmit = () => {
  if (!form.name) { toast.warning('请输入名称'); return }
  if (!form.type) { toast.warning('请选择类型'); return }
  if (!form.host) { toast.warning(form.type === 'sqlite' ? '请输入文件路径' : '请输入主机地址'); return }
  if (form.type !== 'sqlite' && !form.port) { toast.warning('请输入端口'); return }
  if (form.type !== 'sqlite' && !form.username) { toast.warning('请输入用户名'); return }

  if (duplicateInstance.value) {
    toast.warning(`该主机(${form.host})、用户名(${form.username})和端口(${form.port})的数据库连接已存在`)
    return
  }

  loading.value = true

  if (form.isRemote && props.type === 'edit') {
    const passwordChanged = form.password !== '••••••••'
    const submitData = {
      name: form.name,
      type: form.type,
      host: form.host,
      port: parseInt(form.port) || 3306,
      username: form.username,
      password: passwordChanged ? form.password : undefined,
      database: form.database || undefined,
      version: form.version,
      description: form.description,
      ssl: form.ssl
    }
    fetch(`/api/remote-servers/${form.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(submitData)
    })
      .then(res => res.json())
      .then(data => {
        if (data.code === 0) {
          toast.success('更新成功')
          emit('success')
          emit('update:modelValue', false)
        } else {
          toast.error(data.msg || '更新失败')
        }
      })
      .catch(err => {
        toast.error('更新失败: ' + err.message)
      })
      .finally(() => {
        loading.value = false
      })
    return
  }

  const localPasswordChanged = form.password !== '••••••••'
  const submitData = {
    id: form.id,
    name: form.name,
    type: form.type,
    host: form.host,
    port: parseInt(form.port) || 3306,
    username: form.username,
    password: localPasswordChanged ? form.password : undefined,
    database: form.database || undefined,
    version: form.version,
    description: form.description,
    ssl: form.ssl,
    permission: form.permission
  }

  const request = props.type === 'create'
    ? databaseApi.create(submitData)
    : databaseApi.update(submitData)

  request.then(res => {
    if (res.data.code === 0) {
      toast.success(props.type === 'create' ? '创建成功' : '更新成功')
      emit('success')
      emit('update:modelValue', false)
    } else {
      const detail = res.data.msg || (props.type === 'create' ? '创建失败' : '更新失败')
      toast.error(detail)
    }
  }).catch(err => {
    toast.error((props.type === 'create' ? '创建失败' : '更新失败') + ': ' + (err.message || '网络错误'))
  }).finally(() => {
    loading.value = false
  })
}
</script>
