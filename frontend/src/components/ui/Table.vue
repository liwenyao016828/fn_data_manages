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
    return () => h('thead', { class: cn('[&_tr]:border-b', props.class) }, slots)
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
      class: cn('border-t bg-[#F5F5F5] font-medium [&>tr]:last:border-b-0', props.class),
    }, slots)
  },
})

export const TableRow = defineComponent({
  name: 'TableRow',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('tr', {
      class: cn('border-b transition-colors hover:bg-[#FAFAFA] data-[state=selected]:bg-[#F5F5F5]', props.class),
    }, slots)
  },
})

export const TableHead = defineComponent({
  name: 'TableHead',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('th', {
      class: cn(
        'h-10 px-2 text-left align-middle font-medium text-[#8C8C8C] [&:has([role=checkbox])]:pr-0 [&>[role=checkbox]]:translate-y-0.5',
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
        'p-2 align-middle [&:has([role=checkbox])]:pr-0 [&>[role=checkbox]]:translate-y-0.5',
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
</script>
