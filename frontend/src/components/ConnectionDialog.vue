<template>
  <Dialog :open="modelValue" @update:open="onDialogUpdate">
    <DialogContent class="max-w-[520px]">
      <DialogHeader>
        <DialogTitle>连接信息</DialogTitle>
      </DialogHeader>

      <div class="flex flex-col gap-0">
        <div class="mb-6">
          <h3 class="text-sm font-semibold text-foreground mb-3">容器连接</h3>
          <div class="bg-muted/50 border border-border rounded-md p-4">
            <div class="flex items-center mb-3 last:mb-0">
              <span class="w-12 text-sm text-muted-foreground shrink-0">地址</span>
              <div class="flex-1 flex items-center justify-between">
                <span class="text-sm font-medium text-foreground">{{ containerAddress }}</span>
                <Button variant="ghost" size="icon" class="h-7 w-7" @click="copyText(containerAddress)">
                  <Copy class="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
            <div class="flex items-center">
              <span class="w-12 text-sm text-muted-foreground shrink-0">端口</span>
              <div class="flex-1 flex items-center justify-between">
                <span class="text-sm font-medium text-foreground">{{ containerPort }}</span>
                <Button variant="ghost" size="icon" class="h-7 w-7" @click="copyText(containerPort)">
                  <Copy class="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
            <p class="text-xs text-muted-foreground mt-2">PHP 运行环境/容器安装的应用使用此连接地址</p>
          </div>
        </div>

        <div class="mb-6">
          <h3 class="text-sm font-semibold text-foreground mb-3">外部连接</h3>
          <div class="bg-muted/50 border border-border rounded-md p-4">
            <div class="flex items-center mb-3">
              <span class="w-12 text-sm text-muted-foreground shrink-0">地址</span>
              <div class="flex-1 flex items-center justify-between">
                <span class="text-sm font-medium text-foreground">{{ externalAddress }}</span>
                <Button variant="ghost" size="icon" class="h-7 w-7" @click="copyText(externalAddress)">
                  <Copy class="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
            <div class="flex items-center">
              <span class="w-12 text-sm text-muted-foreground shrink-0">端口</span>
              <div class="flex-1 flex items-center justify-between">
                <span class="text-sm font-medium text-foreground">{{ externalPort }}</span>
                <Button variant="ghost" size="icon" class="h-7 w-7" @click="copyText(externalPort)">
                  <Copy class="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
            <p class="text-xs text-muted-foreground mt-2">非容器或外部连接使用此地址</p>
          </div>
        </div>

        <div class="mb-6">
          <div class="flex items-center justify-between">
            <span class="text-sm font-medium text-foreground">远程访问</span>
            <Switch :checked="remoteAccess" disabled />
          </div>
          <p class="text-xs text-muted-foreground mt-2">远程访问功能暂不可用</p>
        </div>

        <div>
          <label class="text-sm font-medium text-foreground">* root 密码</label>
          <div class="flex gap-2 mt-1.5">
            <Input
              v-model="rootPassword"
              :type="showPwd ? 'text' : 'password'"
              placeholder="root密码"
              class="flex-1 bg-amber-50/50 dark:bg-amber-950/20"
            />
            <Button variant="outline" size="icon" @click="showPwd = !showPwd">
              <Eye v-if="!showPwd" class="h-4 w-4" />
              <EyeOff v-else class="h-4 w-4" />
            </Button>
            <Button variant="outline" size="sm" @click="copyPassword">复制</Button>
            <Button variant="outline" size="sm" @click="randomPassword">随机密码</Button>
          </div>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="handleClose">取消</Button>
        <Button @click="handleConfirm">确认</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { toast } from 'vue-sonner'
import { Copy, Eye, EyeOff } from 'lucide-vue-next'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/Dialog.vue'
import { Button } from '@/components/ui/Button.vue'
import { Input } from '@/components/ui/Input.vue'
import { Switch } from '@/components/ui/Switch.vue'

const props = defineProps({
  modelValue: Boolean,
  data: Object
})

const emit = defineEmits(['update:modelValue', 'close', 'saved'])

const onDialogUpdate = (val) => {
  emit('update:modelValue', val)
  if (!val) emit('close')
}

const showPwd = ref(false)
const remoteAccess = ref(false)
const rootPassword = ref('')
const containerAddress = ref('')
const containerPort = ref('3306')
const externalAddress = ref('')
const externalPort = ref('')

watch(() => props.data, (val) => {
  if (val) {
    rootPassword.value = val.password ? '••••••••' : ''
    containerAddress.value = `${val.type}-${val.name}`
    containerPort.value = val.port || (val.type === 'redis' ? 6379 : 3306)
    externalAddress.value = val.host || ''
    externalPort.value = val.port || (val.type === 'redis' ? 6379 : 3306)
  }
}, { immediate: true })

const copyText = (text) => {
  navigator.clipboard.writeText(text).then(() => {
    toast.success('复制成功')
  }).catch(() => {
    toast.error('复制失败')
  })
}

const copyPassword = () => {
  if (rootPassword.value === '••••••••') {
    toast.warning('密码已隐藏，请输入新密码后再复制')
    return
  }
  copyText(rootPassword.value)
}

const randomPassword = () => {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*'
  const array = new Uint32Array(16)
  crypto.getRandomValues(array)
  let result = ''
  for (let i = 0; i < 16; i++) {
    result += chars.charAt(array[i] % chars.length)
  }
  rootPassword.value = result
}

const handleConfirm = () => {
  const db = props.data
  if (!db) {
    emit('update:modelValue', false)
    return
  }

  const updateData = {
    id: db.id,
    name: db.name,
    host: externalAddress.value || db.host,
    port: parseInt(externalPort.value) || db.port,
    username: db.username
  }

  if (rootPassword.value && rootPassword.value !== '••••••••') {
    updateData.password = rootPassword.value
  }

  fetch('/api/databases/db', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updateData)
  })
    .then(r => r.json())
    .then(data => {
      if (data.code === 0) {
        toast.success('保存成功')
        emit('update:modelValue', false)
        emit('saved')
      } else {
        toast.error(data.msg || '保存失败')
      }
    })
    .catch(() => {
      toast.error('保存失败')
    })
}

const handleClose = () => {
  emit('update:modelValue', false)
}
</script>
