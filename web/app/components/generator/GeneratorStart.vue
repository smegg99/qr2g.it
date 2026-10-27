<!-- app/components/generator/GeneratorStart.vue -->
<template>
  <div class="home-start">
    <h1 class="home-brand">
      <img
        src="/qr2g.it.svg"
        :alt="t('app.name')"
        class="home-logo"
        width="320"
        height="105"
      >
    </h1>
    <p class="home-lead">
      {{ t('app.tagline') }}
    </p>
    <RepoInputForm
      v-model="model"
      autofocus
      :loading="stage !== 'idle'"
      :disabled="stage !== 'idle'"
      :show-invalid="!!error"
      @submit="value => emit('submit', value)"
      @input-clear="emit('dismiss')"
    />
    <div :id="TURNSTILE_HOST" />
    <CoverageBar />
    <TopRepositories />
    <CurlHint :repository="model" />
  </div>
</template>

<script setup lang="ts">
import type { Stage } from '~/composables/useGenerator'
import { TURNSTILE_HOST } from '~/composables/useTurnstile'
import type { ApiError } from '~/utils/api'
import type { DesignProblem } from '~/utils/design'
import RepoInputForm from './RepoInputForm.vue'
import CoverageBar from '~/components/top/CoverageBar.vue'
import CurlHint from '~/components/top/CurlHint.vue'
import TopRepositories from '~/components/top/TopRepositories.vue'

const props = defineProps<{ stage: Stage, error: ApiError | null, problems: DesignProblem[] }>()
const model = defineModel<string>({ required: true })
const emit = defineEmits<{
  (e: 'submit', value: string): void
  (e: 'dismiss'): void
}>()
const { t } = useI18n()

const RETRYABLE = new Set(['network', 'github_unavailable', 'github_rate_limited', 'turnstile_unavailable', 'turnstile_failed', 'verification_failed', 'rate_limited', 'unknown'])

// Errors are toasts, so the page never moves; the input keeps its error look until the text changes.
const toast = useToast()
const errorText = useErrorText()
let shown = 0
watch(() => props.error, (error) => {
  if (shown) toast.dismiss(shown)
  shown = 0
  if (!error) return
  shown = toast.error(errorText(error), {
    timeout: 8000,
    action: RETRYABLE.has(error.code) ? { label: t('generator.retry'), onClick: () => emit('submit', model.value) } : undefined,
  })
})
watch(() => props.problems, (problems) => {
  if (!problems.length) return
  toast.warning(`${t('generator.prefill.title')}: ${problems.map(p => t(`generator.prefill.${p.reason}`, { param: p.param })).join(' ')}`, { timeout: 10000 })
}, { immediate: true })
onBeforeUnmount(() => shown && toast.dismiss(shown))
</script>
