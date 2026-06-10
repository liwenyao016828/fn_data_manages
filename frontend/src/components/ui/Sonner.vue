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
          toast: 'group toast group-[.toaster]:bg-white group-[.toaster]:text-[#1A1A1A] group-[.toaster]:border-[#E8E8E8] group-[.toaster]:shadow-lg',
          description: 'group-[.toast]:text-[#8C8C8C]',
          actionButton: 'group-[.toast]:bg-[#1A1A1A] group-[.toast]:text-white',
          cancelButton: 'group-[.toast]:bg-[#F5F5F5] group-[.toast]:text-[#8C8C8C]',
        },
      },
    })
  },
})
</script>
