<script>
import { defineComponent, h } from 'vue'
import { cn } from '@/lib/utils'
import {
  TabsContent as RekaTabsContent,
  TabsList as RekaTabsList,
  TabsRoot,
  TabsTrigger as RekaTabsTrigger,
} from 'reka-ui'

export const Tabs = TabsRoot

export const TabsList = defineComponent({
  name: 'TabsList',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h(RekaTabsList, {
      class: cn(
        'inline-flex h-9 items-center justify-center rounded-xl bg-muted p-1 text-muted-foreground gap-0.5',
        props.class,
      ),
    }, slots)
  },
})

export const TabsTrigger = defineComponent({
  name: 'TabsTrigger',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h(RekaTabsTrigger, {
      class: cn(
        'inline-flex items-center justify-center whitespace-nowrap rounded-lg px-3 py-1 text-sm font-medium ring-offset-background transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow-[0_1px_3px_rgba(0,0,0,0.08)]',
        props.class,
      ),
    }, slots)
  },
})

export const TabsContent = defineComponent({
  name: 'TabsContent',
  props: { class: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h(RekaTabsContent, {
      class: cn(
        'ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 mt-2',
        props.class,
      ),
    }, slots)
  },
})
</script>
