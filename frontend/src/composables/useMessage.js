import { ref } from 'vue'

const messageState = ref({
  show: false,
  type: 'success',
  text: '',
  duration: 3000
})

export function useMessage() {
  const message = ref(messageState.value)

  const showMessage = (type, text, duration = 3000) => {
    messageState.value = {
      show: true,
      type,
      text,
      duration
    }
  }

  const success = (text, duration) => {
    showMessage('success', text, duration)
  }

  const error = (text, duration) => {
    showMessage('error', text, duration)
  }

  const warning = (text, duration) => {
    showMessage('warning', text, duration)
  }

  const info = (text, duration) => {
    showMessage('info', text, duration)
  }

  const close = () => {
    messageState.value.show = false
  }

  return {
    message: messageState,
    showMessage,
    success,
    error,
    warning,
    info,
    close
  }
}
