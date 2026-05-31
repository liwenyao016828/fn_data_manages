<template>
  <Dialog :open="modelValue" @update:open="onDialogUpdate">
    <DialogContent class="max-w-[480px]">
      <DialogHeader>
        <DialogTitle>{{ type === 'create' ? '添加连接' : '编辑连接' }}</DialogTitle>
      </DialogHeader>

      <div class="flex flex-col gap-4">
        <div v-if="type === 'create'" class="flex flex-col gap-1.5">
          <label class="text-sm font-medium">选择已检测到的实例</label>
          <Select v-model="selectedDetect" @update:model-value="onSelectDetect">
            <SelectTrigger class="w-full">
              <SelectValue placeholder="选择已检测到的实例自动填充" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="inst in detects"
                :key="inst.host + ':' + inst.port"
                :value="inst"
              >
                <span class="flex items-center gap-2">
                  {{ inst.name }}
                  <span class="text-xs text-muted-foreground">{{ inst.host }}:{{ inst.port }}</span>
                  <Badge
                    v-if="inst.status === '已认证'"
                    variant="secondary"
                    class="bg-emerald-500/10 text-emerald-600 border-emerald-200 text-[10px] px-1.5 py-0"
                  >
                    已认证
                  </Badge>
                  <Badge
                    v-else
                    variant="outline"
                    class="bg-amber-500/10 text-amber-600 border-amber-200 text-[10px] px-1.5 py-0"
                  >
                    需密码
                  </Badge>
                </span>
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-sm font-medium">名称 <span class="text-destructive">*</span></label>
          <Input v-model="form.name" placeholder="数据库实例名称" :disabled="type === 'edit'" />
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-sm font-medium">类型 <span class="text-destructive">*</span></label>
          <div class="flex gap-1">
            <Button
              :variant="form.type === 'mysql' ? 'default' : 'outline'"
              size="sm"
              @click="onTypeChange('mysql')"
            >
              MySQL
            </Button>
            <Button
              :variant="form.type === 'redis' ? 'default' : 'outline'"
              size="sm"
              @click="onTypeChange('redis')"
            >
              Redis
            </Button>
          </div>
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-sm font-medium">主机地址 <span class="text-destructive">*</span></label>
          <Input v-model="form.host" placeholder="localhost 或 127.0.0.1" />
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-sm font-medium">端口 <span class="text-destructive">*</span></label>
          <Input v-model="form.port" type="number" min="1" max="65535" />
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-sm font-medium">用户名 <span class="text-destructive">*</span></label>
          <Input v-model="form.username" placeholder="root" />
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-sm font-medium">密码</label>
          <div class="flex gap-2">
            <Input v-model="form.password" :type="showPwd ? 'text' : 'password'" placeholder="数据库密码" class="flex-1" />
            <Button variant="outline" size="sm" @click="showPwd = !showPwd">
              <Eye v-if="!showPwd" class="h-4 w-4" />
              <EyeOff v-else class="h-4 w-4" />
            </Button>
            <Button variant="outline" size="sm" @click="randomPassword">随机</Button>
          </div>
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-sm font-medium">权限 <span class="text-destructive">*</span></label>
          <Select v-model="form.permission">
            <SelectTrigger class="w-full">
              <SelectValue placeholder="选择权限" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="%">所有人(%)</SelectItem>
              <SelectItem value="127.0.0.1">本地服务器(127.0.0.1)</SelectItem>
              <SelectItem value="ip">指定IP</SelectItem>
            </SelectContent>
          </Select>
          <Input
            v-if="form.permission === 'ip'"
            v-model="form.specifyIp"
            placeholder="请输入IP地址，多个IP用逗号分隔"
            class="mt-1"
          />
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-sm font-medium">描述信息</label>
          <Textarea v-model="form.description" placeholder="可选" :rows="2" />
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="handleClose">取消</Button>
        <Button @click="handleSubmit" :disabled="loading">
          <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
          确认
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { reactive, watch, ref } from 'vue'
import { toast } from 'vue-sonner'
import { Eye, EyeOff, Loader2 } from 'lucide-vue-next'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/Dialog.vue'
import { Button } from '@/components/ui/Button.vue'
import { Input } from '@/components/ui/Input.vue'
import { Textarea } from '@/components/ui/Textarea.vue'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/Select.vue'
import { Badge } from '@/components/ui/Badge.vue'
import { databaseApi } from '../api/database'

const props = defineProps({
  modelValue: Boolean,
  type: String,
  data: Object
})

const emit = defineEmits(['update:modelValue', 'close', 'success'])

const loading = ref(false)
const showPwd = ref(false)
const selectedDetect = ref(null)
const detects = ref([])
const detectsLoading = ref(false)

const form = reactive({
  id: 0,
  name: '',
  type: 'mysql',
  username: '',
  password: '',
  permission: '%',
  specifyIp: '',
  description: '',
  host: 'localhost',
  port: 3306,
  database: '',
  ssl: false,
  version: ''
})

const onDialogUpdate = (val) => {
  emit('update:modelValue', val)
  if (!val) emit('close')
  if (val && props.type === 'create') {
    fetchDetects()
  }
}

const fetchDetects = () => {
  detectsLoading.value = true
  fetch('/api/databases/detect')
    .then(r => r.json())
    .then(data => {
      detectsLoading.value = false
      if (data.code === 0) {
        detects.value = data.data || []
      }
    })
    .catch(() => {
      detectsLoading.value = false
    })
}

const onTypeChange = (type) => {
  form.type = type
  if (type === 'redis') {
    form.port = 6379
    form.version = ''
  } else {
    form.port = 3306
    form.version = ''
  }
}

const resetForm = () => {
  form.id = 0
  form.name = ''
  form.type = 'mysql'
  form.username = ''
  form.password = ''
  form.permission = '%'
  form.specifyIp = ''
  form.description = ''
  form.host = 'localhost'
  form.port = 3306
  form.database = ''
  form.ssl = false
  form.version = ''
  selectedDetect.value = null
}

const onSelectDetect = (inst) => {
  if (!inst) return
  const typeLabel = inst.type === 'mysql' ? 'MySQL' : 'Redis'
  const rand = Math.floor(Math.random() * 9000 + 1000)
  form.name = `${typeLabel}_${rand}`
  form.type = inst.type
  form.host = inst.host
  form.port = inst.port
  form.username = inst.username || 'root'
  form.password = inst.password || ''
  form.version = inst.version || ''
  form.description = `自动检测 - ${inst.source}`
}

watch(() => props.data, (val) => {
  if (val) {
    form.id = val.id || 0
    form.name = val.name || ''
    form.type = val.type || 'mysql'
    form.username = val.username || ''
    form.password = val.password || ''
    form.permission = val.permission || '%'
    form.specifyIp = val.specifyIp || ''
    form.description = val.description || ''
    form.host = val.host || 'localhost'
    form.port = val.port || (val.type === 'redis' ? 6379 : 3306)
    form.database = val.database || ''
    form.ssl = val.ssl || false
    form.version = val.version || ''
  }
}, { immediate: true })

watch(() => props.modelValue, (val) => {
  if (val && props.type === 'create') {
    resetForm()
  }
})

const randomPassword = () => {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let result = ''
  for (let i = 0; i < 16; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  form.password = result
}

const handleClose = () => {
  emit('update:modelValue', false)
}

const validateForm = () => {
  if (!form.name) { toast.warning('请输入数据库名'); return false }
  if (!form.type) { toast.warning('请选择类型'); return false }
  if (!form.host) { toast.warning('请输入主机地址'); return false }
  if (!form.port) { toast.warning('请输入端口'); return false }
  if (!form.username) { toast.warning('请输入用户名'); return false }
  if (!form.permission) { toast.warning('请选择权限'); return false }
  return true
}

const handleSubmit = () => {
  if (!validateForm()) return

  loading.value = true
  const submitData = {
    id: form.id,
    name: form.name,
    type: form.type,
    host: form.host,
    port: form.port,
    username: form.username,
    password: form.password,
    database: form.database,
    ssl: form.ssl,
    description: form.description,
    permission: form.permission,
    version: form.version
  }

  const request = props.type === 'create'
    ? databaseApi.create(submitData)
    : databaseApi.update(submitData)

  request.then(res => {
    if (res.data.code === 0) {
      toast.success(props.type === 'create' ? '创建成功' : '保存成功')
      emit('success')
      emit('update:modelValue', false)
    } else {
      toast.error(res.data.msg || '操作失败')
    }
  }).catch(err => {
    toast.error('操作失败: ' + err.message)
  }).finally(() => {
    loading.value = false
  })
}
</script>
