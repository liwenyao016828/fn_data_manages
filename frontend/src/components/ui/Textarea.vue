<script>
import { defineComponent, h } from 'vue'
import { cn } from '@/lib/utils'

export const Textarea = defineComponent({
  name: 'Textarea',
  props: {
    class: { type: String, default: '' },
    modelValue: { type: String, default: '' },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    function onInput(event) {
      emit('update:modelValue', event.target.value)
    }
    return () => h('textarea', {
      value: props.modelValue,
      class: cn(
        'flex min-h-[60px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm text-foreground shadow-sm placeholder:text-helper focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      ),
      onInput,
    })
  },
})
</script>
