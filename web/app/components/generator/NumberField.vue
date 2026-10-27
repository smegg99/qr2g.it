<!-- app/components/generator/NumberField.vue -->
<template>
  <v-number-input
    :id="id"
    v-model="value"
    :aria-label="label"
    :min="min"
    :max="max"
    :step="step"
    :suffix="suffix"
    :disabled="disabled"
    density="compact"
    hide-details
    @blur="clamp"
  />
</template>

<script setup lang="ts">
const props = defineProps<{
  id: string
  label: string
  modelValue: number
  min?: number
  max?: number
  step?: number
  suffix?: string
  disabled?: boolean
}>()

const emit = defineEmits<{ (e: 'update:modelValue', v: number): void }>()

const value = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', Number(v)),
})

function clamp() {
  if (typeof props.min === 'number' && value.value < props.min) value.value = props.min
  if (typeof props.max === 'number' && value.value > props.max) value.value = props.max
}
</script>
