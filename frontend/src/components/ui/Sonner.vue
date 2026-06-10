<script>
import { defineComponent, h } from 'vue'
import { cn } from '@/lib/utils'
import { Toaster as Sonner } from 'vue-sonner'

export const Toaster = defineComponent({
  name: 'Toaster',
  props: {
    class: { type: String, default: '' },
    richColors: { type: Boolean, default: false },
    expand: { type: Boolean, default: false },
    position: {
      type: String,
      default: 'top-center',
      validator: (v) => [
        'top-left', 'top-right', 'bottom-left', 'bottom-right',
        'top-center', 'bottom-center',
      ].includes(v),
    },
  },
  setup(props) {
    return () => h(Sonner, {
      class: cn('fixed z-[9999]', props.class),
      richColors: props.richColors,
      expand: props.expand,
      position: props.position,
      toastOptions: {
        classNames: {
          toast: 'group toast group-[.toaster]:bg-card group-[.toaster]:text-foreground group-[.toaster]:border-border group-[.toaster]:shadow-lg',
          description: 'group-[.toast]:text-muted-foreground',
          actionButton: 'group-[.toast]:bg-primary group-[.toast]:text-primary-foreground',
          cancelButton: 'group-[.toast]:bg-muted group-[.toast]:text-muted-foreground',
        },
      },
    })
  },
})
</script>
