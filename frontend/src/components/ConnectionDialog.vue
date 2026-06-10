<template>
  <Dialog :open="modelValue" @update:open="onDialogUpdate">
    <DialogContent class="max-w-[520px]">
      <DialogHeader>
        <DialogTitle>连接信息</DialogTitle>
      </DialogHeader>

      <div class="flex flex-col gap-0">
        <div class="mb-6">
          <h3 class="text-[12px] font-semibold mb-3" style="color: var(--text-primary)">容器连接</h3>
          <div class="rounded-xl p-4" style="background: var(--muted); border: 1px solid var(--border-subtle)">
            <div class="flex items-center mb-3 last:mb-0">
              <span class="w-12 text-[12px] shrink-0" style="color: var(--text-tertiary)">地址</span>
              <div class="flex-1 flex items-center justify-between">
                <span class="text-[13px] font-medium" style="color: var(--text-primary)">{{ containerAddress }}</span>
                <button class="btn-ghost h-6 w-6 p-0 flex items-center justify-center" @click="copyText(containerAddress)">
                  <Copy class="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
            <div class="flex items-center">
              <span class="w-12 text-[12px] shrink-0" style="color: var(--text-tertiary)">端口</span>
              <div class="flex-1 flex items-center justify-between">
                <span class="text-[13px] font-medium" style="color: var(--text-primary)">{{ containerPort }}</span>
                <button class="btn-ghost h-6 w-6 p-0 flex items-center justify-center" @click="copyText(containerPort)">
                  <Copy class="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
            <p class="text-[11px] mt-2" style="color: var(--text-tertiary)">PHP 运行环境/容器安装的应用使用此连接地址</p>
          </div>
        </div>

        <div class="mb-6">
          <h3 class="text-[12px] font-semibold mb-3" style="color: var(--text-primary)">外部连接</h3>
          <div class="rounded-xl p-4" style="background: var(--muted); border: 1px solid var(--border-subtle)">
            <div class="flex items-center mb-3">
              <span class="w-12 text-[12px] shrink-0" style="color: var(--text-tertiary)">地址</span>
              <div class="flex-1 flex items-center justify-between">
                <span class="text-[13px] font-medium" style="color: var(--text-primary)">{{ externalAddress }}</span>
                <button class="btn-ghost h-6 w-6 p-0 flex items-center justify-center" @click="copyText(externalAddress)">
                  <Copy class="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
            <div class="flex items-center">
              <span class="w-12 text-[12px] shrink-0" style="color: var(--text-tertiary)">端口</span>
              <div class="flex-1 flex items-center justify-between">
                <span class="text-[13px] font-medium" style="color: var(--text-primary)">{{ externalPort }}</span>
                <button class="btn-ghost h-6 w-6 p-0 flex items-center justify-center" @click="copyText(externalPort)">
                  <Copy class="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
            <p class="text-[11px] mt-2" style="color: var(--text-tertiary)">非容器或外部连接使用此地址</p>
          </div>
        </div>

        <div class="mb-6">
          <div class="flex items-center justify-between">
            <span class="text-[12px] font-medium" style="color: var(--text-primary)">远程访问</span>
            <Switch :checked="remoteAccess" disabled />
          </div>
          <p class="text-[11px] mt-2" style="color: var(--text-tertiary)">远程访问功能暂不可用</p>
        </div>

        <div>
          <label class="text-[12px] font-medium" style="color: var(--text-primary)">* root 密码</label>
          <div class="flex gap-2 mt-1.5">
            <Input
              v-model="rootPassword"
              :type="showPwd ? 'text' : 'password'"
              placeholder="root密码"
              class="flex-1"
              style="background: var(--warning-soft)"
            />
            <button class="btn-secondary h-8 w-8 p-0 flex items-center justify-center" @click="showPwd = !showPwd">
              <Eye v-if="!showPwd" class="h-4 w-4" />
              <EyeOff v-else class="h-4 w-4" />
            </button>
            <button class="btn-secondary h-8 text-[11px] px-2" @click="copyPassword">复制</button>
            <button class="btn-secondary h-8 text-[11px] px-2" @click="randomPassword">随机密码</button>
          </div>
        </div>
      </div>

      <div class="flex justify-end gap-2 mt-4">
        <button class="btn-ghost" @click="handleClose">取消</button>
        <button class="btn-primary" @click="handleConfirm">确认</button>
      </div>
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
