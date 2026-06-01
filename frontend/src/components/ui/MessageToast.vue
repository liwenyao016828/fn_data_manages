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
    default: return Info
  }
}

const getStyles = () => {
  switch (localMessage.value.type) {
    case 'success':
      return {
        bg: 'bg-emerald-50',
        border: 'border-emerald-200',
        text: 'text-emerald-700',
        icon: 'text-emerald-500'
      }
    case 'error':
      return {
        bg: 'bg-red-50',
        border: 'border-red-200',
        text: 'text-red-700',
        icon: 'text-red-500'
      }
    case 'warning':
      return {
        bg: 'bg-amber-50',
        border: 'border-amber-200',
        text: 'text-amber-700',
        icon: 'text-amber-500'
      }
    default:
      return {
        bg: 'bg-slate-50',
        border: 'border-slate-200',
        text: 'text-slate-700',
        icon: 'text-slate-500'
      }
  }
}
</script>

<template>
  <Transition name="message">
    <div
      v-if="localMessage.show"
      :class="[
        'fixed top-20 left-1/2 -translate-x-1/2 z-50 px-4 py-3 rounded-lg border shadow-lg flex items-center gap-3 min-w-[280px] max-w-[400px]',
        getStyles().bg,
        getStyles().border,
        getStyles().text
      ]"
    >
      <component
        :is="getIcon()"
        :class="['h-4 w-4 shrink-0', getStyles().icon]"
      />
      <span class="text-sm font-medium flex-1">{{ localMessage.text }}</span>
      <button
        class="p-1 rounded hover:bg-black/5 transition-colors"
        @click="close"
      >
        <X class="h-3.5 w-3.5 opacity-60 hover:opacity-100" />
      </button>
    </div>
  </Transition>
</template>

<style scoped>
.message-enter-active,
.message-leave-active {
  transition: opacity 0.2s ease;
}

.message-enter-from,
.message-leave-to {
  opacity: 0;
}
</style>
