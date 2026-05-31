<script>
import { defineComponent, h } from 'vue'
import { cn } from '@/lib/utils'

export const Card = defineComponent({
  name: 'Card',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('div', {
      class: cn('rounded-xl border border-border bg-card text-card-foreground shadow', props.class),
    }, slots.default?.())
  },
})

export const CardHeader = defineComponent({
  name: 'CardHeader',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('div', {
      class: cn('flex flex-col gap-y-1.5 p-6', props.class),
    }, slots.default?.())
  },
})

export const CardTitle = defineComponent({
  name: 'CardTitle',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('div', {
      class: cn('font-semibold leading-none tracking-tight', props.class),
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
</script>
