<script>
import { defineComponent, computed, h } from "vue"
import { Primitive } from "reka-ui"
import { cva } from "class-variance-authority"
import { cn } from "@/lib/utils"

export const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap font-medium transition-all duration-200 cursor-pointer disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0 outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 active:scale-[0.97] relative",
  {
    variants: {
      variant: {
        default:
          "bg-primary text-primary-foreground hover:bg-primary/90 shadow-[0_1px_2px_rgba(0,0,0,0.05)] hover:shadow-[0_4px_12px_rgba(0,0,0,0.08)]",
        destructive:
          "bg-destructive text-white hover:bg-destructive/90 shadow-[0_1px_2px_rgba(0,0,0,0.05)] hover:shadow-[0_4px_12px_rgba(0,0,0,0.08)] focus-visible:ring-destructive/20",
        outline:
          "border border-border bg-transparent text-foreground hover:bg-muted hover:border-border-strong",
        secondary:
          "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        ghost: "text-foreground hover:bg-muted shadow-none",
        link: "text-primary underline-offset-4 hover:underline shadow-none",
        primary:
          "bg-primary text-primary-foreground hover:bg-primary/90 shadow-[0_1px_2px_rgba(0,0,0,0.05)] hover:shadow-[0_4px_12px_rgba(0,0,0,0.08)]",
        accent:
          "bg-primary text-primary-foreground hover:bg-primary/90 shadow-[0_1px_2px_rgba(0,0,0,0.05)] hover:shadow-[0_4px_12px_rgba(0,0,0,0.08)]",
      },
      size: {
        default: "h-9 px-4 py-2 text-sm rounded-[var(--btn-radius)]",
        sm: "h-8 px-3 text-xs rounded-[var(--btn-radius)]",
        lg: "h-10 px-6 text-sm rounded-[var(--btn-radius)]",
        icon: "h-9 w-9 rounded-[var(--btn-radius)]",
        xs: "h-7 px-2 text-[12px] rounded-[calc(var(--btn-radius)-2px)]",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

export const Button = defineComponent({
  name: 'Button',
  props: {
    variant: { type: String, default: 'default' },
    size: { type: String, default: 'default' },
    asChild: { type: Boolean, default: false },
    as: { type: String, default: 'button' },
    class: { type: String, default: '' },
    loading: { type: Boolean, default: false },
    disabled: { type: Boolean, default: false },
  },
  setup(props, { slots, attrs }) {
    const delegatedProps = computed(() => {
      const { class: _, loading: __, disabled: ___, ...delegated } = props
      return delegated
    })
    return () => {
      const isDisabled = props.disabled || props.loading
      const spinner = props.loading
        ? h('span', { class: 'absolute inset-0 flex items-center justify-center' }, [
            h('span', { class: 'btn-spinner' })
          ])
        : null
      const contentWrapper = h('span', {
        class: cn('contents', props.loading && 'invisible')
      }, slots.default?.())

      return h(
        Primitive,
        {
          ...delegatedProps.value,
          ...attrs,
          as: props.asChild ? 'template' : props.as,
          'as-child': props.asChild,
          class: cn(buttonVariants({ variant: props.variant, size: props.size }), props.class),
          disabled: isDisabled,
        },
        () => [spinner, contentWrapper]
      )
    }
  },
})
</script>
