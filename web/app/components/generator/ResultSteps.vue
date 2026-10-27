<!-- app/components/generator/ResultSteps.vue -->
<!-- The two result steps as one icon button group; the names are the buttons' labels and tooltips. -->
<template>
  <nav
    class="keys"
    :aria-label="t('result.stepsLabel')"
  >
    <button
      v-for="step in STEPS"
      :key="step.value"
      type="button"
      :class="['key', 'steps__key', { 'key--on': model === step.value }]"
      :aria-current="model === step.value ? 'step' : undefined"
      :aria-label="t(`result.steps.${step.value}`)"
      :title="t(`result.steps.${step.value}`)"
      @click="model = step.value"
    >
      <v-icon
        :icon="step.icon"
        size="18"
      />
    </button>
  </nav>
</template>

<script setup lang="ts">
export type ResultStep = 'link' | 'qr'
const STEPS: { value: ResultStep, icon: string }[] = [
  { value: 'link', icon: 'mdi-link-variant' },
  { value: 'qr', icon: 'mdi-qrcode' },
]

const model = defineModel<ResultStep>({ required: true })
const { t } = useI18n()
</script>

<style scoped>
.steps__key { display: inline-flex; align-items: center; justify-content: center; width: 32px; min-width: 32px; height: 32px; padding: 0; }
@media (max-width: 600px) {
  .steps__key { width: 48px; min-width: 48px; height: 40px; }
  .steps__key :deep(.v-icon) { font-size: 22px; }
}
</style>
