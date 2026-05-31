<script>
import { defineComponent, h } from 'vue'
import { cn } from '@/lib/utils'
import { Separator as SeparatorPrimitive } from 'reka-ui'

export const Separator = defineComponent({
  name: 'Separator',
  props: {
    class: { type: String, default: '' },
    orientation: { type: String, default: 'horizontal' },
    decorative: { type: Boolean, default: true },
  },
  setup(props) {
    return () => h(SeparatorPrimitive, {
      orientation: props.orientation,
      decorative: props.decorative,
      class: cn(
        'shrink-0 bg-border',
        props.orientation === 'horizontal' ? 'h-px w-full' : 'h-full w-px',
        props.class,
      ),
    })
  },
})
</script>
