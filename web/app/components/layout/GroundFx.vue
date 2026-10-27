<!-- app/components/layout/GroundFx.vue -->
<template>
  <div
    class="ground"
    :data-scene="scene.name"
    :style="washes"
    aria-hidden="true"
  >
    <canvas
      ref="canvas"
      :class="['ground__canvas', { 'ground__canvas--ready': ready }]"
    />
    <div class="ground__veil" />
  </div>
</template>

<script setup lang="ts">
import type { NeatGradient } from '@firecms/neat'
import { useTheme } from 'vuetify'
import { groundConfig, pickScene, sceneWashes } from '~/theme/ground'

// A WebGL gradient over CSS washes; the washes stay when WebGL is missing or the build fails.
const theme = useTheme()
const canvas = ref<HTMLCanvasElement | null>(null)
const ready = ref(false)
const still = ref(false)
const scene = pickScene()

let gradient: NeatGradient | null = null
let build = 0
let links: MutationObserver | null = null
let motion: MediaQueryList | null = null

const key = computed(() => {
  const colors = theme.current.value.colors
  return [theme.current.value.dark, colors.background, still.value].join()
})

const washes = computed(() => {
  const [a, b, c] = sceneWashes(scene, theme.current.value.dark)
  return { '--ground-a': a, '--ground-b': b, '--ground-c': c }
})

function supportsWebgl() {
  try {
    const probe = document.createElement('canvas')
    return Boolean(probe.getContext('webgl2') || probe.getContext('webgl'))
  }
  catch {
    return false
  }
}

// Neat 0.x appends an attribution link next to the canvas; it is unreachable and read by screen readers.
function scrub() {
  canvas.value?.parentElement
    ?.querySelectorAll('a[data-n], a[href="https://neat.firecms.co"]')
    .forEach(link => link.remove())
}

function drop() {
  try {
    gradient?.destroy()
  }
  catch {
    // Decorative; a failed teardown must not break the page.
  }
  gradient = null
  ready.value = false
}

async function start() {
  const el = canvas.value
  if (!el || !supportsWebgl()) return
  const id = ++build
  drop()
  try {
    const { NeatGradient } = await import('@firecms/neat')
    if (id !== build || canvas.value !== el) return
    const current = theme.current.value
    gradient = new NeatGradient({ ref: el, ...groundConfig(scene, current.dark, String(current.colors.background), still.value) })
    scrub()
    ready.value = true
  }
  catch {
    drop()
  }
}

function onMotion() {
  still.value = Boolean(motion?.matches)
}

onMounted(() => {
  motion = window.matchMedia('(prefers-reduced-motion: reduce)')
  motion.addEventListener('change', onMotion)
  onMotion()
  const root = canvas.value?.parentElement
  if (root) {
    links = new MutationObserver(scrub)
    links.observe(root, { childList: true, subtree: true })
  }
  void start()
})

watch(key, () => void start())

onBeforeUnmount(() => {
  build++
  motion?.removeEventListener('change', onMotion)
  links?.disconnect()
  drop()
})
</script>

<style scoped>
.ground {
  position: fixed;
  inset: 0;
  z-index: 0;
  overflow: hidden;
  pointer-events: none;
  background:
    radial-gradient(60% 50% at 18% 20%, color-mix(in srgb, var(--ground-a) 30%, transparent), transparent 70%),
    radial-gradient(50% 45% at 82% 30%, color-mix(in srgb, var(--ground-b) 25%, transparent), transparent 70%),
    radial-gradient(55% 50% at 60% 90%, color-mix(in srgb, var(--ground-c) 20%, transparent), transparent 70%),
    rgb(var(--v-theme-background));
}

.ground__canvas {
  position: absolute;
  inset: 0;
  display: block;
  width: 100%;
  height: 100%;
  opacity: 0;
  transition: opacity 1.4s ease;
}

.ground__canvas--ready {
  opacity: var(--ground-strength);
}

/* Keep the header and page edges close to the theme background. */
.ground__veil {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(to bottom, rgba(var(--v-theme-background), var(--ground-veil-top)), transparent 22%),
    linear-gradient(to top, rgba(var(--v-theme-background), var(--ground-veil-bottom)), transparent 30%);
}

.v-theme--dark .ground {
  --ground-strength: 0.55;
  --ground-veil-top: 0.55;
  --ground-veil-bottom: 0.7;
}

.v-theme--light .ground {
  --ground-strength: 0.48;
  --ground-veil-top: 0.5;
  --ground-veil-bottom: 0.6;
}
</style>
