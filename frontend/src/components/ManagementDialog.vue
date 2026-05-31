<template>
  <InstanceDialog v-model="dialogVisible" :type="type" :data="data" :existing-instances="existingInstances" @success="onSuccess" />
</template>

<script setup>
import { computed } from 'vue'
import InstanceDialog from './InstanceDialog.vue'

const props = defineProps({
  modelValue: Boolean,
  type: String,
  data: Object,
  existingInstances: { type: Array, default: () => [] }
})

const emit = defineEmits(['update:modelValue', 'success'])

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const onSuccess = () => {
  emit('success')
}
</script>
