<script>
import { defineComponent, h } from 'vue'
import { cn } from '@/lib/utils'
import {
  SelectContent as RekaSelectContent,
  SelectGroup as RekaSelectGroup,
  SelectItem as RekaSelectItem,
  SelectItemIndicator,
  SelectItemText,
  SelectLabel as RekaSelectLabel,
  SelectPortal,
  SelectRoot,
  SelectScrollDownButton as RekaSelectScrollDownButton,
  SelectScrollUpButton as RekaSelectScrollUpButton,
  SelectSeparator as RekaSelectSeparator,
  SelectTrigger as RekaSelectTrigger,
  SelectValue as RekaSelectValue,
  SelectViewport,
} from 'reka-ui'

export const Select = SelectRoot

export { RekaSelectGroup as SelectGroup }

export { RekaSelectValue as SelectValue }

export const SelectTrigger = defineComponent({
  name: 'SelectTrigger',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h(RekaSelectTrigger, {
      class: cn(
        'flex h-9 w-full items-center justify-between whitespace-nowrap rounded-[10px] border border-border bg-transparent px-3 py-2 text-sm text-foreground shadow-none ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-primary disabled:cursor-not-allowed disabled:opacity-50 [&>span]:line-clamp-1 cursor-pointer transition-colors hover:border-border-strong',
        props.class,
      ),
    }, {
      default: () => [
        slots.default?.(),
        h(SelectIcon, null, slots.icon),
      ],
    })
  },
})

const SelectIcon = defineComponent({
  name: 'SelectIcon',
  setup(_, { slots }) {
    return () => h('svg', {
      xmlns: 'http://www.w3.org/2000/svg',
      width: '24',
      height: '24',
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2',
      'stroke-linecap': 'round',
      'stroke-linejoin': 'round',
      class: 'h-4 w-4 opacity-50 icon-secondary',
    }, [
      h('path', { d: 'm6 9 6 6 6-6' }),
    ])
  },
})

export const SelectScrollUpButton = defineComponent({
  name: 'SelectScrollUpButton',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h(RekaSelectScrollUpButton, {
      class: cn('flex cursor-default items-center justify-center py-1', props.class),
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
        class: 'h-4 w-4 icon-muted',
      }, [
        h('path', { d: 'm18 15-6-6-6 6' }),
      ]),
    ])
  },
})

export const SelectScrollDownButton = defineComponent({
  name: 'SelectScrollDownButton',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h(RekaSelectScrollDownButton, {
      class: cn('flex cursor-default items-center justify-center py-1', props.class),
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
        class: 'h-4 w-4 icon-muted',
      }, [
        h('path', { d: 'm6 9 6 6 6-6' }),
      ]),
    ])
  },
})

export const SelectContent = defineComponent({
  name: 'SelectContent',
  props: {
    class: { type: String, default: '' },
    position: { type: String, default: 'popper' },
  },
  setup(props, { slots }) {
    return () => h(SelectPortal, null, {
      default: () => [
        h(RekaSelectContent, {
          position: props.position,
          class: cn(
            'relative z-50 max-h-96 min-w-32 overflow-hidden rounded-xl border border-border bg-popover text-popover-foreground shadow-[0_10px_40px_rgba(0,0,0,0.1)] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[side=bottom]:slide-in-from-top-1 data-[side=left]:slide-in-from-right-1 data-[side=right]:slide-in-from-left-1 data-[side=top]:slide-in-from-bottom-1',
            props.position === 'popper' && 'data-[side=bottom]:translate-y-1 data-[side=left]:-translate-x-1 data-[side=right]:translate-x-1 data-[side=top]:-translate-y-1',
            props.class,
          ),
        }, {
          default: () => [
            h(SelectScrollUpButton, null),
            h(SelectViewport, {
              class: cn('p-1', props.position === 'popper' && 'w-full min-w-[var(--reka-select-trigger-width)]'),
            }, slots),
            h(SelectScrollDownButton, null),
          ],
        }),
      ],
    })
  },
})

export const SelectLabel = defineComponent({
  name: 'SelectLabel',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h(RekaSelectLabel, {
      class: cn('px-2 py-1.5 text-sm font-semibold', props.class),
    }, slots)
  },
})

export const SelectSeparator = defineComponent({
  name: 'SelectSeparator',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h(RekaSelectSeparator, {
      class: cn('-mx-1 my-1 h-px bg-muted', props.class),
    }, slots)
  },
})

export const SelectItem = defineComponent({
  name: 'SelectItem',
  props: { class: { type: String, default: '' }, value: { type: String, required: true }, disabled: { type: Boolean, default: false } },
  setup(props, { slots }) {
    return () => h(RekaSelectItem, {
      value: props.value,
      disabled: props.disabled,
      class: cn(
        'relative flex w-full cursor-pointer select-none items-center rounded-lg py-1.5 pl-2 pr-8 text-sm text-foreground outline-none focus:bg-accent/10 focus:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
        props.class,
      ),
    }, {
      default: () => [
        h('span', { class: 'absolute right-2 flex h-3.5 w-3.5 items-center justify-center' }, [
          h(SelectItemIndicator, null, () => [
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
              h('path', { d: 'M20 6 9 17l-5-5' }),
            ]),
          ]),
        ]),
        h(SelectItemText, null, slots),
      ],
    })
  },
})
</script>
