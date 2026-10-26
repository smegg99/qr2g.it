<!-- app/components/home/QrCanvas.vue -->
<!-- eslint-disable vue/no-v-html -->
<template>
  <figure class="preview">
    <div class="preview__well">
      <div
        class="preview__code"
        :style="canvasStyle"
      >
        <Transition
          name="swap"
          mode="out-in"
        >
          <div
            v-if="snapshot"
            key="ready"
            class="preview__svg"
            v-html="snapshot.svg"
          />
          <div
            v-else
            key="loading"
            class="preview__svg"
          >
            <v-progress-circular
              color="accent"
              indeterminate
              size="32"
              width="2"
            />
          </div>
        </Transition>
      </div>
    </div>
    <figcaption class="preview__caption">
      <span>{{ t('home.qr.preview') }}</span>
      <span v-if="snapshot">{{ snapshot.pixelSize }} x {{ snapshot.pixelSize }} {{ t('home.qr.controls.px') }}</span>
    </figcaption>
  </figure>
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
.preview__code {
  position: relative;
  width: 100%;
  aspect-ratio: 1;
  overflow: hidden;
  background: var(--qr-canvas-bg);
  transition: border-radius 0.2s ease;
}

.preview__svg {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.preview__svg :deep(svg) {
  display: block;
  width: 100%;
  height: 100%;
  shape-rendering: crispEdges;
}
</style>
