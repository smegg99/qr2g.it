<!-- app/components/common/KeyGroup.vue -->
<template>
  <div
    class="keys"
    role="group"
    :aria-label="label"
  >
    <button
      v-for="item in items"
      :key="item.value"
      type="button"
      :class="['key', { 'key--on': choice && item.value === model }]"
      :aria-pressed="choice ? item.value === model : undefined"
      :aria-label="item.aria"
      :title="item.title"
      :disabled="disabled"
      @click="pick(item.value)"
    >
      {{ item.label }}
    </button>
  </div>
</template>

<script setup lang="ts">
// Keys sharing borders. `choice` keeps one pressed, as a segmented control; otherwise each key is an action.
export interface KeyItem {
  value: string
  label: string
  title?: string
  aria?: string
}

const props = defineProps<{
  items: KeyItem[]
  label: string
  choice?: boolean
  disabled?: boolean
}>()
const model = defineModel<string>()
const emit = defineEmits<{ (e: 'press', value: string): void }>()

function pick(value: string) {
  if (props.choice) model.value = value
  emit('press', value)
}
</script>
