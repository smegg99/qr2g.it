<!-- app/components/home/QrSection.vue -->
<template>
  <PaneHead :title="t('home.qr.title')">
    <template #meta>
      <span v-if="snapshot">{{ snapshot.modules }} x {{ snapshot.modules }} {{ t('home.qr.controls.modules') }}</span>
    </template>
    <template #actions>
      <ExportBar
        :busy="exporting"
        @export="onExport"
      />
    </template>
  </PaneHead>
  <div class="pane-body qr-body">
    <QrCanvas
      :snapshot="snapshot"
      :background="background"
      :corner-radius="cornerRadius"
      :ready="!!snapshot"
    />
    <QrControls
      v-model:foreground="foreground"
      v-model:background="background"
      v-model:size="size"
      v-model:margin="margin"
      v-model:error-correction="errorCorrection"
      v-model:corner-radius="cornerRadius"
    />
  </div>
</template>

<script setup lang="ts">
import PaneHead from '~/components/common/PaneHead.vue'
import QrCanvas from './QrCanvas.vue'
import QrControls from './QrControls.vue'
import ExportBar from './ExportBar.vue'
import type { QrErrorLevel, QrOptions } from '~/composables/useQrCode'

const props = defineProps<{
  text: string
  filenameHint: string
}>()

const { t } = useI18n()
const { snapshot, build, exportSvg, exportPng, exportJpg, downloadBlob } = useQrCode()

const foreground = ref('#111111')
const background = ref('#ffffff')
const size = ref(512)
const margin = ref(4)
const cornerRadius = ref(0)
const errorCorrection = ref<QrErrorLevel>('M')

const opts = computed<QrOptions>(() => ({
  text: props.text,
  size: size.value,
  margin: margin.value,
  foreground: foreground.value,
  background: background.value,
  errorCorrection: errorCorrection.value,
  cornerRadius: cornerRadius.value,
}))

watch(opts, async (v) => {
  await build({ ...v })
}, { immediate: true })

const exporting = ref(false)

async function onExport(format: 'svg' | 'png' | 'jpg') {
  if (exporting.value) return
  exporting.value = true
  try {
    const safeName = sanitizeFilename(props.filenameHint || 'qr2g')
    const filename = `${safeName}.${format}`
    const exporters = {
      svg: exportSvg,
      png: exportPng,
      jpg: exportJpg,
    } as const
    const blob = await exporters[format]({ ...opts.value })
    downloadBlob(blob, filename)
  }
  finally {
    exporting.value = false
  }
}

function sanitizeFilename(name: string): string {
  return name
    .replace(/^https?:\/\//i, '')
    .replace(/\/$/, '')
    .replace(/\//g, '-')
    .replace(/[^a-z0-9.\-_]/gi, '_')
}
</script>
