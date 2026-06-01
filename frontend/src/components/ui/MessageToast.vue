<script setup>
import { ref, watch } from 'vue'
import { CheckCircle, XCircle, AlertCircle, Info, X } from 'lucide-vue-next'

const props = defineProps({
  message: {
    type: Object,
    default: () => ({ show: false, type: 'success', text: '', duration: 3000 })
  }
})

const emit = defineEmits(['close'])

const localMessage = ref({ ...props.message })

watch(() => props.message, (newVal) => {
  localMessage.value = { ...newVal }
  if (newVal.show && newVal.duration > 0) {
    setTimeout(() => {
      localMessage.value.show = false
      emit('close')
    }, newVal.duration)
  }
}, { deep: true })

const close = () => {
  localMessage.value.show = false
  emit('close')
}

const getIcon = () => {
  switch (localMessage.value.type) {
    case 'success': return CheckCircle
    case 'error': return XCircle
    case 'warning': return AlertCircle
    case 'info': return Info
    default: return Info
  }
}

const getBadgeClass = () => {
  switch (localMessage.value.type) {
    case 'success': return 'badge-status badge-status-success'
    case 'error': return 'badge-status badge-status-error'
    case 'warning': return 'badge-status badge-status-warning'
    case 'info': return 'badge-status badge-status-info'
    default: return 'badge-status badge-status-neutral'
  }
}

const getIconClass = () => {
  switch (localMessage.value.type) {
    case 'success': return 'icon-success'
    case 'error': return 'icon-error'
    case 'warning': return 'icon-warning'
    case 'info': return 'icon-info'
    default: return 'icon-secondary'
  }
}

const getTextClass = () => {
  switch (localMessage.value.type) {
    case 'success': return 'text-success'
    case 'error': return 'text-error'
    case 'warning': return 'text-warning'
    case 'info': return 'text-info'
    default: return 'text-body'
  }
}
</script>

<template>
  <Transition name="message">
    <div
      v-if="localMessage.show"
      :class="['toast-card', getBadgeClass(), 'shadow-lg flex items-center gap-3 min-w-[280px] max-w-[400px]']"
      role="status"
      aria-live="polite"
    >
      <component
        :is="getIcon()"
        :class="['h-4 w-4 shrink-0', getIconClass()]"
      />
      <span :class="['text-sm font-medium flex-1', getTextClass()]">{{ localMessage.text }}</span>
      <button
        class="p-1 rounded hover:bg-black/10 dark:hover:bg-white/10 transition-colors icon-muted"
        @click="close"
        aria-label="关闭"
      >
        <X class="h-3.5 w-3.5" />
      </button>
    </div>
  </Transition>
</template>

<style scoped>
.toast-card {
  position: fixed;
  top: 5rem;
  left: 50%;
  transform: translateX(-50%);
  z-index: 50;
  padding: 0.75rem 1rem;
  border-radius: 0.5rem;
  border-width: 1px;
  backdrop-filter: blur(8px);
}
.message-enter-active,
.message-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.message-enter-from,
.message-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(-8px);
}
</style>
