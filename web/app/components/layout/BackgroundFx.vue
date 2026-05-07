<!-- app/components/layout/BackgroundFx.vue -->
<template>
  <div class="bg-fx" aria-hidden="true" :style="rootStyle">
    <canvas ref="canvas" class="bg-fx__canvas" />
    <div class="bg-fx__fade bg-fx__fade--top" />
    <div class="bg-fx__fade bg-fx__fade--bottom" />
  </div>
</template>

<script setup lang="ts">
import { useTheme } from 'vuetify'
import { useDailyAccentHex } from '~/composables/useMaterialAccentTheme'
import { createBackgroundFxConfig } from '~/theme/backgroundFx'
import { buildMaterialAccentColors } from '~/theme/materialAccent'

type NeatGradientLike = {
  destroy: () => void
  yOffset?: number
}

type NeatGradientConstructor = new (
  options: Record<string, unknown>
) => NeatGradientLike

const NEAT_LINK_SELECTOR = 'a[data-n], a[href="https://neat.firecms.co"]'

const theme = useTheme()
const canvas = ref<HTMLCanvasElement | null>(null)
const accentHex = useDailyAccentHex()

const isDark = computed(() => theme.global.current.value.dark)
const themeColors = computed(() => theme.global.current.value.colors)
const materialColors = computed(() =>
  buildMaterialAccentColors(accentHex.value, isDark.value),
)
const gradientConfig = computed(() =>
  createBackgroundFxConfig({
    accentHex: accentHex.value,
    background: themeColors.value.background as string,
    colors: materialColors.value,
    isDark: isDark.value,
  }),
)
const rootStyle = computed(() => ({
  '--bg-fx-fade-top-opacity': isDark.value ? '0.5' : '0.6',
  '--bg-fx-fade-bottom-opacity': isDark.value ? '0.7' : '0.7',
}))

let gradient: NeatGradientLike | null = null
let NeatGradientClass: NeatGradientConstructor | null = null
let neatLinkObserver: MutationObserver | null = null
let mounted = false
let rebuildFrame = 0
let buildId = 0

function onScroll() {
  if (!gradient) return
  gradient.yOffset = window.scrollY
}

function supportsWebgl() {
  try {
    // Important: do NOT test on the real visible canvas.
    const testCanvas = document.createElement('canvas')

    return Boolean(
      testCanvas.getContext('webgl2')
      || testCanvas.getContext('webgl')
      || testCanvas.getContext('experimental-webgl'),
    )
  }
  catch {
    return false
  }
}

function destroyGradient() {
  try {
    gradient?.destroy()
  }
  catch {
    // The gradient is decorative; failed cleanup should not interrupt the app.
  }
  finally {
    gradient = null
    removeNeatLinks()
  }
}

async function build() {
  if (!mounted) return

  const el = canvas.value
  if (!el) return

  const currentBuildId = ++buildId

  if (!supportsWebgl()) {
    destroyGradient()
    return
  }

  destroyGradient()

  try {
    if (!NeatGradientClass) {
      const mod = await import('@firecms/neat')
      NeatGradientClass = mod.NeatGradient as unknown as NeatGradientConstructor
    }

    // Prevent stale async imports from creating old gradients after theme/HMR changes.
    if (!mounted || currentBuildId !== buildId || canvas.value !== el) return

    gradient = new NeatGradientClass({
      ref: el,
      ...gradientConfig.value,
    })

    removeNeatLinks()
    onScroll()
  }
  catch {
    destroyGradient()
  }
}

function queueBuild() {
  if (!mounted) return

  if (rebuildFrame) {
    cancelAnimationFrame(rebuildFrame)
  }

  rebuildFrame = requestAnimationFrame(() => {
    rebuildFrame = 0
    void build()
  })
}

function removeNeatLinks(root = canvas.value?.parentElement) {
  root
    ?.querySelectorAll(NEAT_LINK_SELECTOR)
    .forEach(link => link.remove())
}

function startNeatLinkGuard() {
  const root = canvas.value?.parentElement
  if (!root) return

  stopNeatLinkGuard()
  removeNeatLinks(root)

  neatLinkObserver = new MutationObserver(() => {
    removeNeatLinks(root)
  })
  neatLinkObserver.observe(root, { childList: true, subtree: true })
}

function stopNeatLinkGuard() {
  neatLinkObserver?.disconnect()
  neatLinkObserver = null
}

onMounted(async () => {
  mounted = true

  await nextTick()

  startNeatLinkGuard()
  await build()

  window.addEventListener('scroll', onScroll, { passive: true })
})

watch(gradientConfig, () => {
  queueBuild()
})

onBeforeUnmount(() => {
  mounted = false
  buildId++

  if (rebuildFrame) {
    cancelAnimationFrame(rebuildFrame)
    rebuildFrame = 0
  }

  window.removeEventListener('scroll', onScroll)

  destroyGradient()
  stopNeatLinkGuard()
})

if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    mounted = false
    buildId++

    if (rebuildFrame) {
      cancelAnimationFrame(rebuildFrame)
      rebuildFrame = 0
    }

    window.removeEventListener('scroll', onScroll)

    destroyGradient()
    stopNeatLinkGuard()
  })
}
</script>

<style scoped>
.bg-fx {
  position: fixed;
  inset: 0;
  z-index: 0;
  overflow: hidden;
  pointer-events: none;
  border-radius: 0;
  opacity: 0.82;
}

.bg-fx__canvas {
  position: absolute;
  inset: 0;
  display: block;
  width: 100%;
  height: 100%;
  border-radius: 0;
  opacity: 0.88;
}

.bg-fx__fade {
  position: absolute;
  left: 0;
  right: 0;
  border-radius: 0;
  pointer-events: none;
}

.bg-fx__fade--top {
  top: 0;
  height: 180px;
  background: linear-gradient(to bottom,
      rgba(var(--v-theme-background), var(--bg-fx-fade-top-opacity)),
      transparent);
}

.bg-fx__fade--bottom {
  bottom: 0;
  height: 240px;
  background: linear-gradient(to top,
      rgba(var(--v-theme-background), var(--bg-fx-fade-bottom-opacity)),
      transparent);
}

.bg-fx :deep(a[data-n]),
.bg-fx :deep(a[href="https://neat.firecms.co"]) {
  display: none !important;
  visibility: hidden !important;
  opacity: 0 !important;
  pointer-events: none !important;
}
</style>
