<script>
import { defineComponent, h } from 'vue'
import { cn } from '@/lib/utils'
import {
  TooltipArrow as RekaTooltipArrow,
  TooltipContent as RekaTooltipContent,
  TooltipPortal as RekaTooltipPortal,
  TooltipProvider as RekaTooltipProvider,
  TooltipRoot as RekaTooltipRoot,
  TooltipTrigger as RekaTooltipTrigger,
} from 'reka-ui'

export const TooltipProvider = RekaTooltipProvider

export const Tooltip = RekaTooltipRoot

export const TooltipTrigger = RekaTooltipTrigger

export const TooltipPortal = RekaTooltipPortal

export const TooltipContent = defineComponent({
  name: 'TooltipContent',
  props: {
    class: { type: String, default: '' },
    sideOffset: { type: Number, default: 4 },
  },
  setup(props, { slots }) {
    return () => h(RekaTooltipContent, {
      sideOffset: props.sideOffset,
      class: cn(
        'z-50 overflow-hidden rounded-md bg-primary px-3 py-1.5 text-xs text-primary-foreground animate-in fade-in-0 zoom-in-95 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2',
        props.class,
      ),
    }, {
      default: () => [
        slots.default?.(),
        h(RekaTooltipArrow, {
          class: 'fill-primary',
        }),
      ],
    })
  },
})
</script>
