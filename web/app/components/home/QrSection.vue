<!-- app/components/home/QrSection.vue -->
<template>
  <v-sheet color="transparent" class="d-flex flex-column ga-5">
    <div class="text-caption text-medium-emphasis">
      {{ t('home.qr.title') }}
    </div>

    <v-row no-gutters class="qr-section__row">
      <v-col cols="12" md="5">
        <QrCanvas :snapshot="snapshot" :background="background" :corner-radius="cornerRadius" :ready="!!snapshot" />
      </v-col>
      <v-col cols="12" md="7">
        <div class="d-flex flex-column ga-5">
          <QrControls v-model:foreground="foreground" v-model:background="background" v-model:size="size"
            v-model:margin="margin" v-model:error-correction="errorCorrection" />
          <ExportBar v-model:corner-radius="cornerRadius" :busy="exporting" @export="onExport" />
        </div>
      </v-col>
    </v-row>
  </v-sheet>
</template>

<script setup lang="ts">
import QrCanvas from './QrCanvas.vue'
import QrControls from './QrControls.vue'
import ExportBar from './ExportBar.vue'
import type { QrErrorLevel, QrOptions } from '~/composables/useQrCode'

const props = defineProps<{
  text: string
  filenameHint: string
}>()

const { t } = useI18n()
const settings = useSettingsStore()
const { snapshot, build, exportSvg, exportPng, exportJpg, downloadBlob } = useQrCode()

const foreground = ref(settings.resolvedTheme === 'dark' ? '#e0e3ec' : '#0e1015')
const background = ref(settings.resolvedTheme === 'dark' ? '#0e1015' : '#ffffff')
const size = ref(512)
const margin = ref(2)
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

<style scoped>
.qr-section__row {
  gap: 24px 0;
}

@media (min-width: 960px) {
  .qr-section__row>.v-col:first-child {
    padding-right: 16px;
  }

  .qr-section__row>.v-col:last-child {
    padding-left: 16px;
  }
}
</style>
