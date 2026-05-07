<!-- app/components/home/QrCanvas.vue -->
<!-- eslint-disable vue/no-v-html -->
<template>
  <div class="d-flex flex-column ga-3">
    <v-sheet class="qr-canvas" rounded="0" :style="canvasStyle">
      <v-fade-transition mode="out-in">
        <div v-if="snapshot" key="ready" class="qr-canvas__svg" v-html="snapshot.svg" />
        <div v-else key="loading" class="qr-canvas__placeholder">
          <v-progress-circular color="primary" indeterminate size="36" width="2" />
        </div>
      </v-fade-transition>
    </v-sheet>

    <div class="d-flex align-baseline justify-space-between ga-2">
      <div class="text-caption text-medium-emphasis">
        {{ t('home.qr.preview') }}
      </div>
      <v-fade-transition>
        <div v-if="snapshot" class="text-caption text-medium-emphasis">
          {{ snapshot.modules }}×{{ snapshot.modules }} {{ t('home.qr.controls.modules') }}
        </div>
      </v-fade-transition>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { QrSnapshot } from '~/composables/useQrCode'

const props = defineProps<{
  snapshot: QrSnapshot | null
  background: string
  cornerRadius: number
  ready: boolean
}>()

const { t } = useI18n()

const canvasStyle = computed(() => ({
  '--qr-canvas-bg': props.background,
  'borderRadius': `${props.cornerRadius}%`,
}))
</script>

<style scoped>
.qr-canvas {
  position: relative;
  aspect-ratio: 1;
  overflow: hidden;
  background: var(--qr-canvas-bg, rgb(var(--v-theme-surface)));
  transition: border-radius 0.2s ease, box-shadow 0.4s ease;
}

.qr-canvas__svg,
.qr-canvas__placeholder {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.qr-canvas__svg :deep(svg) {
  display: block;
  width: 100%;
  height: 100%;
  shape-rendering: crispEdges;
  image-rendering: pixelated;
  image-rendering: crisp-edges;
}
</style>
