<!-- app/components/generator/RepoInputForm.vue -->
<template>
  <v-form
    :class="['repo-bar', { 'repo-bar--error': showInvalid }]"
    :aria-busy="loading || undefined"
    @submit.prevent="onSubmit"
  >
    <v-icon
      icon="mdi-github"
      class="repo-bar__icon"
      aria-hidden="true"
    />
    <input
      id="repository-input"
      ref="inputEl"
      v-model="value"
      class="repo-bar__input"
      :placeholder="t('home.input.placeholder')"
      :aria-label="t('home.input.label')"
      :aria-invalid="showInvalid || undefined"
      :disabled="disabled"
      autocomplete="off"
      autocapitalize="off"
      autocorrect="off"
      spellcheck="false"
      enterkeyhint="go"
    >
    <v-btn
      v-if="value && !disabled"
      icon="mdi-close"
      variant="text"
      size="small"
      class="repo-bar__clear"
      :aria-label="t('home.input.clear')"
      :title="t('home.input.clear')"
      @click="onClear"
    />
    <v-btn
      type="submit"
      color="accent"
      variant="flat"
      class="repo-bar__go"
      :disabled="!canSubmit || disabled"
      :loading="loading"
      :aria-label="t('home.input.submit')"
    >
      <span class="repo-bar__go-text">{{ t('home.input.submit') }}</span>
      <v-icon
        icon="mdi-qrcode"
        class="repo-bar__go-icon"
      />
    </v-btn>
  </v-form>
</template>

<script setup lang="ts">
import { parseRepoInput } from '~/utils/repository'

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
const inputEl = ref<HTMLInputElement | null>(null)
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
  inputEl.value?.focus()
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
