<script>
import { defineComponent, h } from 'vue'
import { cn } from '@/lib/utils'

export const Table = defineComponent({
  name: 'Table',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('div', {
      class: cn('relative w-full overflow-auto', props.class),
    }, [
      h('table', { class: 'w-full caption-bottom text-sm' }, slots),
    ])
  },
})

export const TableHeader = defineComponent({
  name: 'TableHeader',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('thead', { class: cn('[&_tr]:border-b border-border-subtle', props.class) }, slots)
  },
})

export const TableBody = defineComponent({
  name: 'TableBody',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('tbody', { class: cn('[&_tr:last-child]:border-0', props.class) }, slots)
  },
})

export const TableFooter = defineComponent({
  name: 'TableFooter',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('tfoot', {
      class: cn('border-t border-border-subtle bg-muted/50 font-medium [&>tr]:last:border-b-0', props.class),
    }, slots)
  },
})

export const TableRow = defineComponent({
  name: 'TableRow',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('tr', {
      class: cn('border-b border-border-subtle transition-colors hover:bg-accent/5 data-[state=selected]:bg-accent/10', props.class),
    }, slots)
  },
})

export const TableHead = defineComponent({
  name: 'TableHead',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('th', {
      class: cn(
        'h-10 px-3 text-left align-middle font-medium text-muted-foreground text-xs uppercase tracking-wider [&:has([role=checkbox])]:pr-0 [&>[role=checkbox]]:translate-y-0.5',
        props.class,
      ),
    }, slots)
  },
})

export const TableCell = defineComponent({
  name: 'TableCell',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('td', {
      class: cn(
        'p-3 align-middle text-sm [&:has([role=checkbox])]:pr-0 [&>[role=checkbox]]:translate-y-0.5',
        props.class,
      ),
    }, slots)
  },
})

export const TableCaption = defineComponent({
  name: 'TableCaption',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('caption', {
      class: cn('mt-4 text-sm text-muted-foreground', props.class),
    }, slots)
  },
})

export const TableEmpty = defineComponent({
  name: 'TableEmpty',
  props: {
    colspan: { type: Number, default: 99 },
    class: { type: String, default: '' },
  },
  setup(props, { slots }) {
    return () => h('tr', { class: cn('hover:bg-transparent', props.class) }, [
      h('td', {
        colspan: props.colspan,
        class: 'py-12 text-center text-muted-foreground',
      }, [
        h('div', { class: 'flex flex-col items-center justify-center' }, slots.default?.()),
      ]),
    ])
  },
})
</script>
