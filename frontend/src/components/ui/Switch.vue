<script>
import { defineComponent, computed, h } from 'vue'
import { cn } from '@/lib/utils'

export const Switch = defineComponent({
  name: 'Switch',
  props: {
    modelValue: { type: Boolean, default: false },
    disabled: { type: Boolean, default: false },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    const handleClick = () => {
      if (!props.disabled) {
        emit('update:modelValue', !props.modelValue)
      }
    }
    const switchClass = computed(() => cn(
      'peer inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50',
      props.modelValue ? 'bg-primary' : 'bg-input',
      props.disabled ? 'cursor-not-allowed' : 'cursor-pointer'
    ))
    const thumbClass = computed(() => cn(
      'pointer-events-none block h-4 w-4 rounded-full bg-background shadow-lg ring-0 transition-transform',
      props.modelValue ? 'translate-x-4' : 'translate-x-0'
    ))
    return () => h('button', {
      type: 'button',
      class: switchClass.value,
      disabled: props.disabled,
      role: 'switch',
      'aria-checked': props.modelValue,
      onClick: handleClick,
    }, [
      h('span', { class: thumbClass.value })
    ])
  },
})
</script>