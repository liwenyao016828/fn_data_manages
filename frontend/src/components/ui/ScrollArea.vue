<script>
import { defineComponent, h } from 'vue'
import { cn } from '@/lib/utils'
import {
  ScrollAreaCorner,
  ScrollAreaRoot,
  ScrollAreaScrollbar,
  ScrollAreaThumb,
  ScrollAreaViewport,
} from 'reka-ui'

export const ScrollArea = defineComponent({
  name: 'ScrollArea',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h(ScrollAreaRoot, {
      class: cn('relative overflow-hidden', props.class),
    }, {
      default: () => [
        h(ScrollAreaViewport, { class: 'h-full w-full rounded-[inherit]' }, slots),
        h(ScrollBar),
        h(ScrollAreaCorner),
      ],
    })
  },
})

const ScrollBar = defineComponent({
  name: 'ScrollBar',
  props: {
    class: { type: String, default: '' },
    orientation: { type: String, default: 'vertical' },
  },
  setup(props) {
    return () => h(ScrollAreaScrollbar, {
      orientation: props.orientation,
      class: cn(
        'flex touch-none select-none transition-colors',
        props.orientation === 'vertical' && 'h-full w-2.5 border-l border-l-transparent p-px',
        props.orientation === 'horizontal' && 'h-2.5 flex-col border-t border-t-transparent p-px',
        props.class,
      ),
    }, () => [
      h(ScrollAreaThumb, { class: 'relative flex-1 rounded-full bg-border' }),
    ])
  },
})

export { ScrollBar }
</script>
