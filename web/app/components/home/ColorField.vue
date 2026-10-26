<!-- app/components/home/ColorField.vue -->
<template>
  <v-text-field
    :id="id"
    v-model="hex"
    :aria-label="label"
    density="compact"
    hide-details
    spellcheck="false"
    autocomplete="off"
    @blur="commit"
    @keydown.enter="commit"
  >
    <template #prepend-inner>
      <v-menu
        v-model="open"
        :close-on-content-click="false"
        location="bottom start"
      >
        <template #activator="{ props: menuProps }">
          <button
            v-bind="menuProps"
            type="button"
            :aria-label="label"
            :style="{ background: normalized }"
            class="color-swatch"
          />
        </template>

        <v-card class="color-swatch__menu">
          <v-color-picker
            v-model="pickerValue"
            mode="hex"
            hide-inputs
          />
        </v-card>
      </v-menu>
    </template>
  </v-text-field>
</template>

<script setup lang="ts">
const props = defineProps<{
  id: string
  modelValue: string
  label: string
}>()
const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>()

const HEX_RE = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i

const open = ref(false)
const hex = ref(props.modelValue)

const normalized = computed(() => normalize(props.modelValue))
const pickerValue = computed({
  get: () => normalized.value,
  set: v => commitValue(String(v)),
})

watch(() => props.modelValue, (v) => {
  hex.value = v
})

function normalize(v: string): string {
  if (!v) return '#000000'
  if (HEX_RE.test(v)) {
    if (v.length === 4) {
      return '#' + v.slice(1).split('').map(c => c + c).join('')
    }
    return v.toLowerCase()
  }
  return '#000000'
}

function commit() {
  commitValue(hex.value)
}

function commitValue(value: string) {
  if (!HEX_RE.test(value)) return (hex.value = props.modelValue)
  hex.value = normalize(value)
  emit('update:modelValue', hex.value)
}
</script>

<style scoped>
.color-swatch {
  flex: 0 0 auto;
  width: 22px;
  height: 22px;
  border: 1px solid rgb(var(--v-theme-edge));
  border-radius: 3px;
  cursor: pointer;
}

.color-swatch__menu {
  background: rgba(var(--v-theme-surface), 0.95);
  backdrop-filter: blur(18px) saturate(140%);
  -webkit-backdrop-filter: blur(18px) saturate(140%);
}

.color-swatch__menu :deep(.v-color-picker) {
  background: transparent;
  border-radius: 0;
}
</style>
