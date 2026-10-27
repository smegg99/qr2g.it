<!-- app/components/top/Sparkline.vue -->
<!-- Thirty UTC days of filtered redirects as one line; days without collection leave a gap. -->
<template>
  <figure :class="['spark', { 'spark--fill': fill }]">
    <div
      class="spark__plot"
      :style="fill ? undefined : { height: `${height}px` }"
      @pointermove="point"
      @pointerleave="active = null"
    >
      <svg
        :viewBox="`0 0 ${W} ${H}`"
        preserveAspectRatio="none"
        class="spark__svg"
        role="img"
        :aria-label="summary"
      >
        <defs>
          <linearGradient
            :id="fillId"
            x1="0"
            x2="0"
            y1="0"
            y2="1"
          >
            <stop
              offset="0"
              class="spark__fill-top"
            />
            <stop
              offset="1"
              class="spark__fill-bottom"
            />
          </linearGradient>
        </defs>
        <path
          :d="area"
          :fill="`url(#${fillId})`"
        />
        <path
          :d="line"
          class="spark__line"
        />
        <path
          :d="uncounted"
          class="spark__uncounted"
        />
        <line
          v-if="active !== null"
          :x1="x(active)"
          :x2="x(active)"
          y1="0"
          :y2="H"
          class="spark__cursor"
        />
      </svg>
      <span
        v-for="i in lonely"
        :key="i"
        class="spark__dot spark__dot--lone"
        :style="{ left: `${(x(i) / W) * 100}%`, top: `${(y(slots[i]!) / H) * 100}%` }"
      />
      <span
        v-if="active !== null && slots[active] !== null"
        class="spark__dot"
        :style="{ left: `${(x(active) / W) * 100}%`, top: `${(y(slots[active]!) / H) * 100}%` }"
      />
      <span
        v-if="active !== null"
        class="spark__tip"
        :style="{ left: `${Math.min(Math.max((x(active) / W) * 100, 18), 82)}%` }"
      >{{ tip }}</span>
    </div>
    <figcaption
      v-if="axis"
      class="spark__axis"
    >
      <span>{{ day(0) }}</span>
      <span>{{ t('top.peak', { n: n(busiest) }) }}</span>
      <span>{{ day(DAYS - 1) }}</span>
    </figcaption>
  </figure>
</template>

<script setup lang="ts">
import type { StatsDay } from '~/generated/api/types.gen'

// `fill` grows the plot into the space its parent gives it; the drawing scales, the strokes keep their width.
const props = withDefaults(defineProps<{ days: StatsDay[], label: string, height?: number, axis?: boolean, fill?: boolean }>(), { height: 56, axis: false, fill: false })
const { t, d, n } = useI18n()
const fillId = useId()

const DAYS = 30
const W = 300
const H = computed(() => props.height)
const active = ref<number | null>(null)

// Today and the 29 days before it, in UTC like the server's buckets.
const dates = computed(() => {
  const today = new Date()
  return Array.from({ length: DAYS }, (_, i) => {
    const day = new Date(Date.UTC(today.getUTCFullYear(), today.getUTCMonth(), today.getUTCDate() - (DAYS - 1 - i)))
    return day.toISOString().slice(0, 10)
  })
})
const slots = computed(() => {
  const counts = new Map(props.days.map(day => [day.date, day.redirects]))
  return dates.value.map(date => counts.get(date) ?? null)
})
// The busiest day; the scale never drops below one so an empty month draws a flat line.
const busiest = computed(() => Math.max(0, ...slots.value.map(v => v ?? 0)))
const peak = computed(() => Math.max(1, busiest.value))
const x = (i: number) => (i / (DAYS - 1)) * W
// Two units of headroom keep the line's stroke inside the box at the peak.
const y = (v: number) => H.value - 2 - (v / peak.value) * (H.value - 4)

function segments(close: boolean) {
  let out = ''
  let run: number[] = []
  const flush = () => {
    if (!run.length) return
    out += run.map((i, k) => `${k ? 'L' : 'M'}${x(i)},${y(slots.value[i]!)}`).join('')
    if (close) out += `L${x(run[run.length - 1]!)},${H.value}L${x(run[0]!)},${H.value}Z`
    run = []
  }
  slots.value.forEach((v, i) => (v === null ? flush() : run.push(i)))
  flush()
  return out
}
const line = computed(() => segments(false))
// A counted day with no counted neighbour draws no line, so it gets a dot of its own.
const lonely = computed(() => slots.value.flatMap((v, i) => (v !== null && slots.value[i - 1] == null && slots.value[i + 1] == null ? [i] : [])))
// Days before counting began are unknown, not zero: a faint dashed baseline marks them.
const uncounted = computed(() => {
  let out = ''
  slots.value.forEach((v, i) => {
    if (v !== null) return
    const from = Math.max(0, i - 0.5)
    const to = Math.min(DAYS - 1, i + 0.5)
    out += `M${x(from)},${H.value - 1}L${x(to)},${H.value - 1}`
  })
  return out
})
const area = computed(() => segments(true))

function point(e: PointerEvent) {
  const box = (e.currentTarget as HTMLElement).getBoundingClientRect()
  active.value = Math.min(DAYS - 1, Math.max(0, Math.round(((e.clientX - box.left) / box.width) * (DAYS - 1))))
}

const day = (i: number) => d(new Date(`${dates.value[i]}T00:00:00Z`), 'short')
const tip = computed(() => {
  if (active.value === null) return ''
  const v = slots.value[active.value] ?? null
  return `${day(active.value)}: ${v === null ? t('top.notCollected') : t('top.clicks', { n: n(v) }, v)}`
})
const summary = computed(() => `${props.label}. ${t('top.peak', { n: n(busiest.value) })}`)
</script>

<style scoped>
.spark { margin: 0; }
.spark--fill { display: flex; flex-direction: column; min-height: 72px; }
.spark--fill .spark__plot { flex: 1 1 auto; min-height: 56px; }
.spark__plot { position: relative; cursor: crosshair; touch-action: pan-y; }
.spark__svg { display: block; width: 100%; height: 100%; overflow: visible; }
.spark__line { fill: none; stroke: rgb(var(--v-theme-accent)); stroke-width: 2; stroke-linejoin: round; stroke-linecap: round; vector-effect: non-scaling-stroke; }
.spark__fill-top { stop-color: rgb(var(--v-theme-accent)); stop-opacity: 0.32; }
.spark__fill-bottom { stop-color: rgb(var(--v-theme-accent)); stop-opacity: 0; }
.spark__axis { display: flex; justify-content: space-between; margin-top: 6px; font-size: 11px; color: rgb(var(--v-theme-ink3)); font-variant-numeric: tabular-nums; white-space: nowrap; }
.spark__uncounted { fill: none; stroke: rgb(var(--v-theme-ink3)); stroke-width: 1; stroke-dasharray: 3 4; opacity: 0.6; vector-effect: non-scaling-stroke; }
.spark__dot--lone { box-shadow: 0 0 0 3px color-mix(in srgb, rgb(var(--v-theme-accent)) 25%, transparent); }
.spark__cursor { stroke: rgb(var(--v-theme-ink3)); stroke-width: 1; stroke-dasharray: 2 2; vector-effect: non-scaling-stroke; }
.spark__dot { position: absolute; width: 8px; height: 8px; margin: -4px 0 0 -4px; border-radius: 50%; background: rgb(var(--v-theme-accent)); box-shadow: 0 0 0 2px rgb(var(--v-theme-chrome)); pointer-events: none; }
.spark__tip {
  position: absolute;
  bottom: calc(100% + 4px);
  transform: translateX(-50%);
  padding: 2px 6px;
  border: 1px solid rgb(var(--v-theme-line));
  border-radius: 4px;
  background: rgb(var(--v-theme-chrome2));
  color: rgb(var(--v-theme-ink));
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  pointer-events: none;
}
</style>
