import { ref } from 'vue'

const dialogState = ref({
  show: false,
  title: '',
  text: '',
  confirmText: '确认',
  cancelText: '取消',
  variant: 'warning',
  resolve: null
})

let pendingResolve = null

export function useConfirm() {
  const showConfirm = (options) => {
    return new Promise((resolve) => {
      const config = typeof options === 'string' ? { text: options } : options
      pendingResolve = resolve
      dialogState.value = {
        show: true,
        title: config.title || '请确认',
        text: config.text || '',
        confirmText: config.confirmText || '确认',
        cancelText: config.cancelText || '取消',
        variant: config.variant || 'warning',
        resolve
      }
    })
  }

  const handleConfirm = () => {
    dialogState.value.show = false
    if (pendingResolve) {
      pendingResolve(true)
      pendingResolve = null
    }
  }

  const handleCancel = () => {
    dialogState.value.show = false
    if (pendingResolve) {
      pendingResolve(false)
      pendingResolve = null
    }
  }

  return {
    showConfirm,
    handleConfirm,
    handleCancel,
    dialogState
  }
}
