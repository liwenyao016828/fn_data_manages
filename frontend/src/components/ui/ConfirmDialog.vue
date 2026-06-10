<script setup>
import { AlertTriangle, Info, AlertCircle, X } from 'lucide-vue-next'
import { useConfirm } from '../../composables/useConfirm.js'

const { dialogState, handleConfirm, handleCancel } = useConfirm()

const getIcon = () => {
  switch (dialogState.value.variant) {
    case 'danger': return AlertCircle
    case 'warning': return AlertTriangle
    case 'info': return Info
    default: return AlertTriangle
  }
}

const getIconClass = () => {
  switch (dialogState.value.variant) {
    case 'danger': return 'icon-error'
    case 'warning': return 'icon-warning'
    case 'info': return 'icon-info'
    default: return 'icon-warning'
  }
}
</script>

<template>
  <Transition name="confirm-fade">
    <div v-if="dialogState.show" class="confirm-overlay" @click.self="handleCancel">
      <div class="confirm-card" role="alertdialog" aria-modal="true" :aria-label="dialogState.title">
        <button class="confirm-close" @click="handleCancel" aria-label="关闭">
          <X class="h-4 w-4" />
        </button>
        <div class="confirm-body">
          <component :is="getIcon()" :class="['confirm-icon', getIconClass()]" />
          <h3 class="confirm-title">{{ dialogState.title }}</h3>
          <p v-if="dialogState.text" class="confirm-text">{{ dialogState.text }}</p>
        </div>
        <div class="confirm-actions">
          <button class="btn btn-secondary" @click="handleCancel">
            {{ dialogState.cancelText }}
          </button>
          <button
            :class="['btn', dialogState.variant === 'danger' ? 'btn-danger' : 'btn-primary']"
            @click="handleConfirm"
            autofocus
          >
            {{ dialogState.confirmText }}
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.confirm-overlay {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.45);
  backdrop-filter: blur(2px);
}
.confirm-card {
  position: relative;
  width: 360px;
  max-width: calc(100vw - 2rem);
  background: var(--bg-elevated, white);
  border: 1px solid var(--border-primary, #e5e7eb);
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.15);
  color: var(--text-primary, #1a1a1a);
}
.confirm-close {
  position: absolute;
  top: 12px;
  right: 12px;
  background: transparent;
  border: 0;
  padding: 4px;
  cursor: pointer;
  color: var(--text-tertiary, #6b7280);
  border-radius: 6px;
}
.confirm-close:hover { background: var(--bg-secondary, #f3f4f6); }
.confirm-body { display: flex; flex-direction: column; align-items: center; text-align: center; gap: 8px; }
.confirm-icon { width: 36px; height: 36px; margin-bottom: 4px; }
.confirm-title { font-size: 16px; font-weight: 600; margin: 0; }
.confirm-text { font-size: 13px; color: var(--text-secondary, #4b5563); margin: 0; line-height: 1.6; }
.confirm-actions { display: flex; gap: 8px; margin-top: 20px; justify-content: flex-end; }
.btn {
  padding: 7px 16px;
  font-size: 13px;
  font-weight: 500;
  border-radius: 6px;
  border: 1px solid transparent;
  cursor: pointer;
  transition: opacity 0.15s;
}
.btn:hover { opacity: 0.85; }
.btn-secondary {
  background: var(--bg-secondary, #f3f4f6);
  color: var(--text-primary, #1a1a1a);
  border-color: var(--border-primary, #e5e7eb);
}
.btn-primary {
  background: var(--accent-primary, #2563eb);
  color: white;
}
.btn-danger {
  background: var(--color-error, #dc2626);
  color: white;
}
.confirm-fade-enter-active, .confirm-fade-leave-active {
  transition: opacity 0.15s ease;
}
.confirm-fade-enter-from, .confirm-fade-leave-to { opacity: 0; }
</style>
