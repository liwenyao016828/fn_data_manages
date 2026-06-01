<script>
import { defineComponent, h } from 'vue'
import { cn } from '@/lib/utils'
import {
  DialogClose as RekaDialogClose,
  DialogContent as RekaDialogContent,
  DialogDescription as RekaDialogDescription,
  DialogOverlay as RekaDialogOverlay,
  DialogPortal as RekaDialogPortal,
  DialogRoot,
  DialogTitle as RekaDialogTitle,
  DialogTrigger as RekaDialogTrigger,
} from 'reka-ui'

export const Dialog = DialogRoot

export { RekaDialogTrigger as DialogTrigger }

export { RekaDialogPortal as DialogPortal }

export { RekaDialogClose as DialogClose }

export const DialogOverlay = defineComponent({
  name: 'DialogOverlay',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h(RekaDialogOverlay, {
      class: cn('fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0', props.class),
    }, slots)
  },
})

export const DialogContent = defineComponent({
  name: 'DialogContent',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h(RekaDialogContent, {
      class: cn(
        'fixed left-1/2 top-1/2 z-50 grid w-full max-w-lg -translate-x-1/2 -translate-y-1/2 gap-4 border border-border bg-card p-6 shadow-lg duration-200 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 sm:rounded-xl',
        props.class,
      ),
    }, {
      default: () => [
        slots.default?.(),
        h(RekaDialogClose, {
          class: 'absolute right-4 top-4 rounded-sm opacity-70 transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 disabled:pointer-events-none data-[state=open]:bg-muted data-[state=open]:text-muted-foreground',
        }, () => [
          h('svg', {
            xmlns: 'http://www.w3.org/2000/svg',
            width: '24',
            height: '24',
            viewBox: '0 0 24 24',
            fill: 'none',
            stroke: 'currentColor',
            'stroke-width': '2',
            'stroke-linecap': 'round',
            'stroke-linejoin': 'round',
            class: 'h-4 w-4',
          }, [
            h('path', { d: 'M18 6 6 18' }),
            h('path', { d: 'm6 6 12 12' }),
          ]),
          h('span', { class: 'sr-only' }, 'Close'),
        ]),
      ],
    })
  },
})

export const DialogTitle = defineComponent({
  name: 'DialogTitle',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h(RekaDialogTitle, {
      class: cn('text-lg font-semibold leading-none tracking-tight', props.class),
    }, slots)
  },
})

export const DialogDescription = defineComponent({
  name: 'DialogDescription',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h(RekaDialogDescription, {
      class: cn('text-sm text-muted-foreground', props.class),
    }, slots)
  },
})

export const DialogHeader = defineComponent({
  name: 'DialogHeader',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('div', {
      class: cn('flex flex-col gap-1.5 text-center sm:text-left', props.class),
    }, slots)
  },
})

export const DialogFooter = defineComponent({
  name: 'DialogFooter',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('div', {
      class: cn('flex flex-col-reverse sm:flex-row sm:justify-end sm:gap-2', props.class),
    }, slots)
  },
})
</script>
