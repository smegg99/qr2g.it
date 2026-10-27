<!-- app/components/layout/ScrollRail.vue -->
<!-- The page's scroll position drawn over the content, so a long page never loses width to a native scrollbar. -->
<template>
  <div
    v-show="scrollable"
    class="rail"
    :class="{ 'rail--on': shown }"
    aria-hidden="true"
    @pointerenter="hover = true"
    @pointerleave="hover = false"
  >
    <div
      class="rail__thumb"
      :style="{ height: `${thumb}px`, transform: `translateY(${offset}px)` }"
      @pointerdown="grab"
    />
  </div>
</template>

<script setup lang="ts">
const IDLE_MS = 900
const MIN_THUMB = 32

const scrollable = ref(false)
const thumb = ref(0)
const offset = ref(0)
const moving = ref(false)
const hover = ref(false)
const dragging = ref(false)
const shown = computed(() => moving.value || hover.value || dragging.value)
let idle: ReturnType<typeof setTimeout> | undefined
let observer: ResizeObserver | undefined

function measure() {
  const root = document.documentElement
  const view = window.innerHeight
  const total = root.scrollHeight
  scrollable.value = total > view + 1
  if (!scrollable.value) return
  thumb.value = Math.max(MIN_THUMB, (view / total) * view)
  offset.value = (window.scrollY / (total - view)) * (view - thumb.value)
}

function onScroll() {
  measure()
  moving.value = true
  clearTimeout(idle)
  idle = setTimeout(() => (moving.value = false), IDLE_MS)
}

function grab(e: PointerEvent) {
  const el = e.currentTarget as HTMLElement
  el.setPointerCapture(e.pointerId)
  dragging.value = true
  const startY = e.clientY
  const startScroll = window.scrollY
  const root = document.documentElement
  const ratio = (root.scrollHeight - window.innerHeight) / (window.innerHeight - thumb.value)
  const move = (ev: PointerEvent) => window.scrollTo({ top: startScroll + (ev.clientY - startY) * ratio, behavior: 'instant' })
  const stop = () => {
    dragging.value = false
    el.removeEventListener('pointermove', move)
    el.removeEventListener('pointerup', stop)
    el.removeEventListener('pointercancel', stop)
  }
  el.addEventListener('pointermove', move)
  el.addEventListener('pointerup', stop)
  el.addEventListener('pointercancel', stop)
}

onMounted(() => {
  measure()
  window.addEventListener('scroll', onScroll, { passive: true })
  window.addEventListener('resize', measure, { passive: true })
  observer = new ResizeObserver(measure)
  observer.observe(document.body)
})
onBeforeUnmount(() => {
  clearTimeout(idle)
  observer?.disconnect()
  window.removeEventListener('scroll', onScroll)
  window.removeEventListener('resize', measure)
})
</script>

<style scoped>
.rail { position: fixed; inset: 0 0 0 auto; z-index: 150; width: 10px; opacity: 0; transition: opacity 0.25s ease; }
.rail--on { opacity: 1; }
.rail__thumb {
  position: absolute;
  top: 0;
  right: 2px;
  width: 6px;
  border-radius: 3px;
  background: color-mix(in srgb, rgb(var(--v-theme-ink)) 35%, transparent);
  cursor: grab;
  touch-action: none;
}
.rail--on .rail__thumb:hover, .rail__thumb:active { width: 8px; background: color-mix(in srgb, rgb(var(--v-theme-ink)) 55%, transparent); }
@media (prefers-reduced-motion: reduce) { .rail { transition: none; } }
</style>
