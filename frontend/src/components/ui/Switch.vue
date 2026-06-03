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
      'peer inline-flex h-[22px] w-[40px] shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50',
      props.modelValue ? 'bg-primary' : 'bg-input',
    ))
    const thumbClass = computed(() => cn(
      'pointer-events-none block h-[18px] w-[18px] rounded-full bg-background shadow-[0_1px_3px_rgba(0,0,0,0.15)] ring-0 transition-transform duration-200',
      props.modelValue ? 'translate-x-[18px]' : 'translate-x-0',
    ))
    return () => h('button', {
      type: 'button',
      class: switchClass.value,
      disabled: props.disabled,
      role: 'switch',
      'aria-checked': props.modelValue,
      onClick: handleClick,
    }, [
      h('span', { class: thumbClass.value }),
    ])
  },
})
</script>
