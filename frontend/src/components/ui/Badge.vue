<script>
import { defineComponent, h, computed } from 'vue'
import { cn } from '@/lib/utils'
import { cva } from 'class-variance-authority'

export const badgeVariants = cva(
  'inline-flex items-center rounded-lg border px-2 py-0.5 text-[11px] font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
  {
    variants: {
      variant: {
        default: 'border-transparent bg-primary text-primary-foreground shadow hover:bg-primary/80',
        secondary: 'border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80',
        destructive: 'border-transparent bg-destructive text-white shadow hover:bg-destructive/80',
        outline: 'text-foreground',
        success: 'badge-status badge-status-success border-transparent',
        warning: 'badge-status badge-status-warning border-transparent',
        error: 'badge-status badge-status-error border-transparent',
        info: 'badge-status badge-status-info border-transparent',
        neutral: 'badge-status badge-status-neutral border-transparent',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  },
)

export const Badge = defineComponent({
  name: 'Badge',
  props: {
    variant: {
      type: String,
      default: 'default',
      validator: (v) => ['default', 'secondary', 'destructive', 'outline', 'success', 'warning', 'error', 'info', 'neutral'].includes(v),
    },
    class: { type: String, default: '' },
  },
  setup(props, { slots }) {
    const classes = computed(() => cn(badgeVariants({ variant: props.variant }), props.class))
    return () => h('div', { class: classes.value }, slots.default?.())
  },
})
</script>
