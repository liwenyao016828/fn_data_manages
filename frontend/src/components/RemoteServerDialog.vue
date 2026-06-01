<template>
  <Dialog :open="modelValue" @update:open="onDialogUpdate">
    <DialogContent class="max-w-[560px]">
      <DialogHeader>
        <DialogTitle>{{ type === 'create' ? '添加远程服务器' : '编辑远程服务器' }}</DialogTitle>
      </DialogHeader>

      <div class="flex flex-col gap-4">
        <div class="flex flex-col gap-1.5">
          <label class="text-sm font-medium">名称 <span class="text-destructive">*</span></label>
          <Input v-model="form.name" placeholder="服务器名称" />
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
              :variant="form.type === 'mariadb' ? 'default' : 'outline'"
              size="sm"
              @click="onTypeChange('mariadb')"
            >
              MariaDB
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
          <label class="text-sm font-medium">数据库版本 <span class="text-destructive">*</span></label>
          <div class="flex gap-1">
            <template v-if="form.type === 'mysql' || form.type === 'mariadb'">
              <Button
                :variant="form.version === '8.x' ? 'default' : 'outline'"
                size="sm"
                @click="form.version = '8.x'"
              >
                8.x
              </Button>
              <Button
                :variant="form.version === '5.7' ? 'default' : 'outline'"
                size="sm"
                @click="form.version = '5.7'"
              >
                5.7
              </Button>
              <Button
                :variant="form.version === '5.6' ? 'default' : 'outline'"
                size="sm"
                @click="form.version = '5.6'"
              >
                5.6
              </Button>
            </template>
            <template v-if="form.type === 'redis'">
              <Button
                :variant="form.version === '7.x' ? 'default' : 'outline'"
                size="sm"
                @click="form.version = '7.x'"
              >
                7.x
              </Button>
              <Button
                :variant="form.version === '6.x' ? 'default' : 'outline'"
                size="sm"
                @click="form.version = '6.x'"
              >
                6.x
              </Button>
            </template>
          </div>
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-sm font-medium">数据库地址 <span class="text-destructive">*</span></label>
          <div class="flex gap-2">
            <Input v-model="form.host" placeholder="请输入数据库地址" class="flex-1" />
            <Tooltip>
              <TooltipTrigger as-child>
                <Button variant="ghost" size="icon" @click="form.host = '127.0.0.1'">
                  <Monitor class="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>使用当前服务器地址</TooltipContent>
            </Tooltip>
          </div>
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-sm font-medium">端口 <span class="text-destructive">*</span></label>
          <Input v-model.number="form.port" type="number" placeholder="3306" />
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-sm font-medium">用户名 <span class="text-destructive">*</span></label>
          <Input v-model="form.username" placeholder="root" />
          <p class="text-xs text-muted-foreground">root 用户或者拥有 root 权限的数据库用户</p>
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-sm font-medium">密码</label>
          <div class="flex gap-2">
            <Input v-model="form.password" :type="showPwd ? 'text' : 'password'" placeholder="请输入密码" class="flex-1" />
            <Button variant="outline" size="icon" @click="showPwd = !showPwd">
              <Eye v-if="!showPwd" class="h-4 w-4" />
              <EyeOff v-else class="h-4 w-4" />
            </Button>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <Switch :checked="form.ssl" @update:checked="form.ssl = $event" />
          <label class="text-sm font-medium">使用 SSL</label>
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-sm font-medium">描述信息</label>
          <Textarea v-model="form.description" placeholder="可选" :rows="2" />
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="handleCancel">取消</Button>
        <Button variant="outline" @click="handleTest" :disabled="testing">
          <Loader2 v-if="testing" class="h-4 w-4 animate-spin" />
          连接测试
        </Button>
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
import { Monitor, Eye, EyeOff, Loader2 } from 'lucide-vue-next'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/Dialog.vue'
import { Button } from '@/components/ui/Button.vue'
import { Input } from '@/components/ui/Input.vue'
import { Textarea } from '@/components/ui/Textarea.vue'
import { Switch } from '@/components/ui/Switch.vue'
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/Tooltip.vue'

const props = defineProps({
  modelValue: Boolean,
  type: String,
  data: Object
})

const emit = defineEmits(['update:modelValue', 'close', 'success'])

const loading = ref(false)
const testing = ref(false)
const showPwd = ref(false)

const form = reactive({
  id: 0,
  name: '',
  type: 'mysql',
  version: '',
  host: '',
  port: 3306,
  username: 'root',
  password: '',
  ssl: false,
  description: ''
})

const onDialogUpdate = (val) => {
  emit('update:modelValue', val)
  if (!val) emit('close')
}

const onTypeChange = (type) => {
  form.type = type
  if (type === 'redis') {
    form.port = 6379
    form.version = ''
  } else {
    form.port = 23366
    form.version = ''
  }
}

watch(() => props.data, (val) => {
  if (val) {
    form.id = val.id || 0
    form.name = val.name || ''
    form.type = val.type || 'mysql'
    form.version = val.version || ''
    form.host = val.host || ''
    form.port = val.port || (val.type === 'redis' ? 6379 : 23366)
    form.username = val.username || 'root'
    form.password = (val.password && val.password !== '••••••••') ? val.password : ''
    form.ssl = val.ssl || false
    form.description = val.description || ''
  }
}, { immediate: true })

watch(() => props.modelValue, (val) => {
  if (val && props.type === 'create') {
    form.id = 0
    form.name = ''
    form.type = 'mysql'
    form.version = ''
    form.host = ''
    form.port = 23366
    form.username = 'root'
    form.password = ''
    form.ssl = false
    form.description = ''
  }
})

const handleCancel = () => {
  emit('update:modelValue', false)
}

const validateForm = () => {
  if (!form.name) { toast.warning('请输入名称'); return false }
  if (!form.type) { toast.warning('请选择类型'); return false }
  if (!form.host) { toast.warning('请输入数据库地址'); return false }
  if (!form.port) { toast.warning('请输入端口'); return false }
  if (!form.username) { toast.warning('请输入用户名'); return false }
  return true
}

const handleTest = () => {
  if (!validateForm()) return
  testing.value = true
  fetch(`/api/remote-servers/test`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      type: form.type,
      host: form.host,
      port: form.port,
      username: form.username,
      password: form.password || undefined,
      ssl: form.ssl
    })
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        toast.success('连接成功')
      } else {
        toast.error(data.msg || '连接失败')
      }
    })
    .catch(() => {
      toast.error('连接失败，请检查配置')
    })
    .finally(() => {
      testing.value = false
    })
}

const handleSubmit = () => {
  if (!validateForm()) return

  loading.value = true
  const submitData = {
    id: form.id,
    name: form.name,
    type: form.type,
    version: form.version,
    host: form.host,
    port: form.port,
    username: form.username,
    password: form.password,
    ssl: form.ssl,
    description: form.description
  }

  const url = props.type === 'create'
    ? '/api/remote-servers'
    : `/api/remote-servers/${form.id}`
  const method = props.type === 'create' ? 'POST' : 'PUT'

  fetch(url, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(Object.fromEntries(Object.entries(submitData).filter(([_, v]) => v !== undefined)))
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        toast.success(props.type === 'create' ? '添加成功' : '保存成功')
        emit('success')
        emit('update:modelValue', false)
      } else {
        toast.error(data.msg || '操作失败')
      }
    })
    .catch(() => {
      toast.error('操作失败')
    })
    .finally(() => {
      loading.value = false
    })
}
</script>
