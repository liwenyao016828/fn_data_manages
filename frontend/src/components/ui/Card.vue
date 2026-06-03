<script>
import { defineComponent, h } from 'vue'
import { cn } from '@/lib/utils'

export const Card = defineComponent({
  name: 'Card',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('div', {
      class: cn('rounded-2xl border border-border-subtle bg-card text-card-foreground shadow-[var(--card-shadow)] transition-all duration-200', props.class),
    }, slots.default?.())
  },
})

export const CardHeader = defineComponent({
  name: 'CardHeader',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('div', {
      class: cn('flex flex-col gap-y-1.5 p-6 pb-4 border-b border-border-subtle', props.class),
    }, slots.default?.())
  },
})

export const CardTitle = defineComponent({
  name: 'CardTitle',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('div', {
      class: cn('font-semibold leading-none tracking-tight text-[15px]', props.class),
    }, slots.default?.())
  },
})

export const CardDescription = defineComponent({
  name: 'CardDescription',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('p', {
      class: cn('text-sm text-muted-foreground', props.class),
    }, slots.default?.())
  },
})

export const CardContent = defineComponent({
  name: 'CardContent',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('div', {
      class: cn('p-6 pt-0', props.class),
    }, slots.default?.())
  },
})

export const CardFooter = defineComponent({
  name: 'CardFooter',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('div', {
      class: cn('flex items-center p-6 pt-0', props.class),
    }, slots.default?.())
  },
})

export const CardMeta = defineComponent({
  name: 'CardMeta',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('div', {
      class: cn('flex items-start gap-3', props.class),
    }, [
      slots.icon
        ? h('div', { class: 'flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-accent/10' }, slots.icon())
        : null,
      h('div', { class: 'flex flex-col' }, [
        slots.title ? h('div', { class: 'font-medium text-sm' }, slots.title()) : null,
        slots.description ? h('p', { class: 'text-xs text-muted-foreground' }, slots.description()) : null,
      ]),
    ])
  },
})
</script>
