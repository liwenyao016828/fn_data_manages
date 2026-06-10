<script>
import { defineComponent, h } from 'vue'
import { cn } from '@/lib/utils'

export const Input = defineComponent({
  name: 'Input',
  props: {
    class: { type: String, default: '' },
    type: { type: String, default: 'text' },
    modelValue: { type: [String, Number], default: undefined },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    function onInput(event) {
      emit('update:modelValue', event.target.value)
    }
    return () => h('input', {
      type: props.type,
      value: props.modelValue,
      class: cn(
        'flex h-9 w-full rounded-[10px] border border-border bg-transparent px-3 py-1 text-sm text-foreground transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/30 focus-visible:border-primary disabled:cursor-not-allowed disabled:opacity-50 hover:border-border-strong',
        props.class,
      ),
      onInput,
    })
  },
})
</script>
