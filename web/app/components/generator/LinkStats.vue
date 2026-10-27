<!-- app/components/generator/LinkStats.vue -->
<!-- Clicks on the short link: counters that roll in like a slot machine, and the last 30 days as a chart. -->
<template>
  <!-- One layout from the first paint: the tiles and the chart area are there before the numbers,
       which roll in when they arrive, so the card never changes size. -->
  <div class="clicks">
    <dl class="clicks__tiles">
      <div
        v-for="tile in tiles"
        :key="tile.key"
        :class="['clicks__tile', { 'clicks__tile--lead': tile.key === 'clicks' }]"
      >
        <dt>{{ tile.label }}</dt>
        <dd>
          <span v-if="note">-</span>
          <template v-else>
            <!-- The rolling digits are decoration; assistive technology reads the plain number. -->
            <NumberFlow
              :value="shown(tile.value)"
              :locales="locale"
              :spin-timing="SPIN"
              aria-hidden="true"
            />
            <span class="sr-only clicks__value">{{ n(shown(tile.value)) }}</span>
          </template>
        </dd>
      </div>
      <div class="clicks__tile">
        <dt>{{ t('result.last') }}</dt>
        <dd
          class="clicks__last"
          :class="{ 'clicks__last--on': stats }"
          :title="stats?.lastRedirectAt ? d(new Date(stats.lastRedirectAt), 'medium') : undefined"
        >
          {{ lastClick }}
        </dd>
      </div>
      <div class="clicks__tile clicks__tile--stars">
        <dt>{{ t('result.stars') }}</dt>
        <dd class="clicks__stars">
          <svg
            class="clicks__star"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path d="M12 2.6l2.86 5.8 6.4.93-4.63 4.51 1.09 6.37L12 17.2l-5.72 3.01 1.09-6.37L2.74 9.33l6.4-.93z" />
          </svg>
          <span v-if="link.stars === undefined">-</span>
          <template v-else>
            <NumberFlow
              :value="starsReady ? link.stars : 0"
              :locales="locale"
              :spin-timing="SPIN"
              aria-hidden="true"
            />
            <span class="sr-only">{{ n(link.stars) }}</span>
          </template>
        </dd>
      </div>
    </dl>
    <div class="clicks__chart">
      <Transition name="chart">
        <Sparkline
          v-if="stats && !note"
          :days="stats.days"
          :label="t('result.chartLabel')"
          fill
          axis
        />
      </Transition>
      <p
        v-if="note || (stats && !stats.collecting)"
        :class="['clicks__note', { 'clicks__note--corner': !note }]"
      >
        {{ note || t('result.paused') }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import NumberFlow from '@number-flow/vue'
import { useQuery } from '@pinia/colada'
import Sparkline from '~/components/top/Sparkline.vue'
import { getLinkStatsQuery } from '~/generated/api/@pinia/colada.gen'
import type { Link } from '~/generated/api/types.gen'
import { timeAgo } from '~/utils/time'

const props = defineProps<{ link: Link }>()
const { t, d, n, locale } = useI18n()

// Long, overshooting spins read as a slot machine settling rather than a counter ticking.
const SPIN = { duration: 1400, easing: 'cubic-bezier(0.2, 0.9, 0.25, 1.1)' }

const listing = computed(() => props.link.policy.listing)
const query = useQuery(() => ({
  ...getLinkStatsQuery({ path: { slug: props.link.slug } }),
  enabled: listing.value === 'listed',
}))
const stats = computed(() => query.data.value)

// The counters show zero until the numbers arrive, then roll to them.
const ready = ref(false)
let timer: ReturnType<typeof setTimeout> | undefined
watch(stats, (value) => {
  if (value && !ready.value) timer = setTimeout(() => (ready.value = true), 120)
}, { immediate: true })
// Stars come with the link itself, so they roll in as soon as the card is on screen.
const starsReady = ref(false)
const starsTimer = setTimeout(() => (starsReady.value = true), 120)
onBeforeUnmount(() => {
  clearTimeout(timer)
  clearTimeout(starsTimer)
})
const shown = (value: number | undefined) => (ready.value ? value ?? 0 : 0)

const tiles = computed(() => [
  { key: 'clicks', label: t('result.clicks'), value: stats.value?.total },
  { key: 'month', label: t('result.month'), value: stats.value?.total30 },
  { key: 'week', label: t('result.week'), value: stats.value?.total7 },
])
const lastClick = computed(() => {
  if (note.value) return '-'
  if (!stats.value) return ''
  return stats.value.lastRedirectAt ? timeAgo(stats.value.lastRedirectAt, locale.value, t('top.now')) : t('result.never')
})

const note = computed(() => {
  switch (listing.value) {
    case 'pending': return t('result.pending')
    case 'owner_noindex': return t('result.hidden')
    case 'listed': return ''
    default: return t('result.unavailable')
  }
})
</script>

<style scoped>
.clicks { display: flex; flex-direction: column; gap: 12px; min-height: 0; }
.clicks__tiles { display: grid; grid-template-columns: 1.2fr 1fr 1fr 1.3fr 1fr; gap: 10px; margin: 0; }
.clicks__tile {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, rgb(var(--v-theme-line)) 70%, transparent);
  border-radius: 8px;
  background: color-mix(in srgb, rgb(var(--v-theme-chrome)) 60%, transparent);
}
.clicks__tile dt { font-size: 11px; color: rgb(var(--v-theme-ink3)); white-space: nowrap; }
/* A fixed line height keeps the tile the same before and after the number arrives. */
.clicks__tile dd { height: 27px; margin: 0; font: 700 22px/27px var(--font-mono); color: rgb(var(--v-theme-ink)); font-variant-numeric: tabular-nums; white-space: nowrap; }
.clicks__tile--lead dd { color: rgb(var(--v-theme-accent)); }
.clicks__tile dd.clicks__stars { display: flex; align-items: center; gap: 6px; }
/* GitHub's star amber, with rounded joins so the small star reads as a shape rather than spikes. */
.clicks__star { flex: 0 0 auto; width: 18px; height: 18px; fill: #e3b341; stroke: #e3b341; stroke-width: 1.4; stroke-linejoin: round; }
.clicks__tile dd.clicks__last { overflow: hidden; font: 650 14px/27px var(--s99ab-font-sans); text-overflow: ellipsis; opacity: 0; transition: opacity 0.4s ease; }
.clicks__last--on { opacity: 1 !important; }
.clicks__chart { position: relative; display: flex; flex-direction: column; flex: 1 1 auto; min-height: 94px; }
.clicks__chart > :deep(.spark) { flex: 1 1 auto; }
.clicks__note { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; margin: 0; padding: 0 12px; font-size: 12px; color: rgb(var(--v-theme-ink3)); text-align: center; pointer-events: none; }
.clicks__note--corner { inset: 0 0 auto auto; justify-content: flex-end; padding: 0; }
.chart-enter-active { transition: opacity 0.5s ease; }
.chart-enter-from { opacity: 0; }
/* A phone gets two rows: the three counts, then the last click beside the stars. */
@media (max-width: 600px) {
  .clicks__tiles { grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 8px; }
  .clicks__tile { grid-column: span 2; padding: 8px 10px; }
  .clicks__tile:nth-child(4) { grid-column: span 3; }
  .clicks__tile--stars { grid-column: span 3; }
  .clicks__tile dd { font-size: 18px; }
}
</style>
