<!-- app/components/top/CoverageBar.vue -->
<!-- Links next to GitHub's count of public repositories; GitHub's number is a lower bound.
     The bar holds its place from the first paint, so nothing below it moves when the numbers arrive. -->
<template>
  <figure class="coverage">
    <div
      class="coverage__track"
      role="progressbar"
      :aria-valuenow="data?.mapped"
      aria-valuemin="0"
      :aria-valuemax="data?.publicRepositories"
      :aria-label="text || undefined"
      :aria-busy="!data || undefined"
    >
      <div
        class="coverage__fill"
        :class="{ 'coverage__fill--on': data }"
        :style="{ width: `${share * 100}%` }"
      />
    </div>
    <figcaption
      class="coverage__text"
      :class="{ 'coverage__text--on': data }"
    >
      <template v-if="data">
        {{ text }}
        <span
          v-if="data.publicRepositories"
          class="coverage__share"
        >{{ n(share, { style: 'percent', maximumSignificantDigits: 2 }) }}</span>
      </template>
      <template v-else>
        &nbsp;
      </template>
    </figcaption>
  </figure>
</template>

<script setup lang="ts">
import { useQuery } from '@pinia/colada'
import { getCoverageQuery } from '~/generated/api/@pinia/colada.gen'

const { t, n } = useI18n()
const query = useQuery(() => getCoverageQuery())
const data = computed(() => query.data.value)
const share = computed(() => (data.value?.publicRepositories ? Math.min(1, data.value.mapped / data.value.publicRepositories) : 0))
const text = computed(() => {
  const d = data.value
  if (!d) return ''
  return d.publicRepositories
    ? t('top.coverage', { mapped: n(d.mapped), total: n(d.publicRepositories, { notation: 'compact', maximumFractionDigits: 0 }) })
    : t('top.mapped', { mapped: n(d.mapped) })
})
</script>

<style scoped>
.coverage { margin: 28px 0 0; }
.coverage__track { height: 6px; overflow: hidden; border-radius: 3px; background: color-mix(in srgb, rgb(var(--v-theme-ink)) 12%, transparent); }
/* Today's share is far below one pixel; the minimum keeps the start of the bar visible. */
.coverage__fill { min-width: 0; height: 100%; border-radius: 3px; background: rgb(var(--v-theme-accent)); transition: min-width 0.8s cubic-bezier(0.2, 0.8, 0.2, 1), width 0.8s cubic-bezier(0.2, 0.8, 0.2, 1); }
.coverage__fill--on { min-width: 3px; }
.coverage__text { margin-top: 8px; overflow: hidden; opacity: 0; transition: opacity 0.4s ease; font-size: clamp(10.5px, 3.3vw, 12px); color: rgb(var(--v-theme-ink3)); font-variant-numeric: tabular-nums; text-overflow: ellipsis; white-space: nowrap; }
.coverage__text--on { opacity: 1; }
.coverage__share { margin-left: 6px; font-weight: 650; color: rgb(var(--v-theme-ink2)); }
</style>
