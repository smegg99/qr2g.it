<!-- app/components/generator/AssetPreview.vue -->
<template>
  <figure class="preview">
    <div class="preview__well asset">
      <img
        v-if="src"
        :src="src"
        :alt="alt"
        class="asset__img"
        :class="{ 'asset__img--busy': loading }"
      >
      <v-progress-circular
        v-else-if="loading"
        indeterminate
        color="accent"
        size="28"
        width="2"
      />
    </div>
    <figcaption class="preview__caption">
      <span>{{ t('design.preview') }}</span>
      <span v-if="info">{{ info.width }} x {{ info.height }} px</span>
    </figcaption>
    <Reveal :show="!!error">
      <ErrorMessage
        v-if="shownError"
        :error="shownError"
        :retry="shownError.code !== 'invalid_design'"
        @retry="emit('retry')"
      />
      <p
        v-if="shownError?.details"
        class="asset__details"
      >
        <span
          v-for="(code, field) in shownError.details"
          :key="field"
        >{{ t(`design.invalid.${code}`, { field: t(`design.fields.${field}`) }) }}</span>
      </p>
    </Reveal>
  </figure>
</template>

<script setup lang="ts">
import type { AssetInfo } from '~/composables/useAsset'
import type { ApiError } from '~/utils/api'
import Reveal from '~/components/motion/Reveal.vue'
import ErrorMessage from './ErrorMessage.vue'

const props = defineProps<{
  src: string | null
  info: AssetInfo | null
  loading: boolean
  error: ApiError | null
  alt: string
}>()
const emit = defineEmits<{ (e: 'retry'): void }>()
const { t } = useI18n()
const shownError = useLatched(() => props.error)
</script>

<style scoped>
.asset { display: flex; align-items: center; justify-content: center; aspect-ratio: 1; }
.asset__img { display: block; width: 100%; height: 100%; object-fit: contain; transition: opacity 0.15s ease; animation: asset-in 0.45s ease-out; }
/* Only the first render fades in; later renders swap in place. */
@keyframes asset-in { from { opacity: 0; transform: scale(0.97); } }
.asset__img--busy { opacity: 0.6; }
.asset__details { display: flex; flex-direction: column; margin: 6px 0 0; font-size: 13px; color: rgb(var(--v-theme-error)); }
@media (prefers-reduced-motion: reduce) { .asset__img { transition: none; animation: none; } }
</style>
