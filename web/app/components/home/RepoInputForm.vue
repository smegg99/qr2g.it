<!-- app/components/home/RepoInputForm.vue -->
<template>
  <v-form class="d-flex align-stretch ga-2 w-100" @submit.prevent="onSubmit">
    <v-text-field ref="inputEl" v-model="value" class="flex-1-1-auto" :placeholder="t('home.input.placeholder')"
      :aria-label="t('home.input.label')" :disabled="disabled" :error="showInvalid" autocomplete="off"
      autocapitalize="off" autocorrect="off" :clearable="!disabled" prepend-inner-icon="mdi-github" spellcheck="false"
      density="comfortable" hide-details @click:clear="onClear" @keydown.enter.prevent="onSubmit" />

    <v-btn type="submit" color="primary" icon="mdi-arrow-right" :disabled="!canSubmit || disabled" :loading="loading"
      :aria-label="t('home.input.submit')" class="repo-form__submit" />
  </v-form>
</template>

<script setup lang="ts">
import { parseRepoInput } from '~/composables/useShortLink'

const props = defineProps<{
  modelValue: string
  disabled?: boolean
  loading?: boolean
  showInvalid?: boolean
  autofocus?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue' | 'submit', v: string): void
  (e: 'input-clear'): void
}>()

const { t } = useI18n()
const inputEl = ref<{ focus: () => void } | null>(null)
const value = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v ?? ''),
})

const canSubmit = computed(() => parseRepoInput(value.value) !== null)

watch(value, () => {
  if (props.showInvalid) emit('input-clear')
})

function onClear() {
  emit('update:modelValue', '')
  emit('input-clear')
}

function onSubmit() {
  if (!canSubmit.value || props.disabled || props.loading) return
  emit('submit', value.value.trim())
}

defineExpose({ focus: () => inputEl.value?.focus() })

onMounted(() => {
  if (props.autofocus) inputEl.value?.focus()
})
</script>

<style scoped>
.repo-form__submit {
  height: 48px;
  width: 48px;
  min-width: 48px;
  flex: 0 0 48px;
}
</style>
