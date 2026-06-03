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
        'flex min-h-[60px] w-full rounded-[10px] border border-border bg-transparent px-3 py-2 text-sm text-foreground transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/30 focus-visible:border-primary disabled:cursor-not-allowed disabled:opacity-50 hover:border-border-strong',
        props.class,
      ),
      onInput,
    })
  },
})
</script>
