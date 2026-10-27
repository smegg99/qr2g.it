<!-- app/components/generator/DesignSection.vue -->
<!-- The QR editor and its export: the format keys download what the preview shows. -->
<template>
  <section
    class="design"
    :aria-labelledby="headingId"
  >
    <h2
      :id="headingId"
      class="sr-only"
    >
      {{ t('design.title') }}
    </h2>
    <div class="design__preview">
      <AssetPreview
        :src="preview.src.value"
        :info="preview.info.value"
        :loading="preview.loading.value"
        :error="preview.error.value"
        :alt="t('design.previewAlt', { name: fullName })"
        @retry="preview.refresh()"
      />
    </div>
    <div class="design__side">
      <DesignControls />
      <div
        class="design__export"
        role="group"
        :aria-label="t('exporter.label')"
      >
        <span class="control__label">{{ t('exporter.label') }}</span>
        <div class="design__downloads">
          <v-btn
            v-for="format in FORMATS"
            :key="format"
            :color="format === 'png' ? 'accent' : undefined"
            :variant="format === 'png' ? 'flat' : 'outlined'"
            prepend-icon="mdi-download"
            class="design__download"
            :disabled="!formats.includes(format) || (!!pending && pending !== format)"
            :loading="pending === format"
            :title="t('exporter.download', { format: format.toUpperCase() })"
            @click="download(format)"
          >
            {{ format.toUpperCase() }}
          </v-btn>
        </div>
        <v-btn
          variant="text"
          :prepend-icon="designCopied ? 'mdi-check' : 'mdi-link-variant'"
          :color="designCopied ? 'success' : undefined"
          class="design__share"
          :title="t('exporter.designLinkNote')"
          @click="copyDesign(designUrl)"
        >
          {{ designCopied ? t('exporter.copiedDesign') : t('exporter.copyDesign') }}
        </v-btn>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { Link } from '~/generated/api/types.gen'
import { useAssetExport, useAssetPreview } from '~/composables/useAsset'
import { useDesignStore } from '~/stores/design'
import { FORMATS, assetFilename, designLink, formatsFor, type Format } from '~/utils/design'
import AssetPreview from './AssetPreview.vue'
import DesignControls from './DesignControls.vue'

const props = defineProps<{ link: Link }>()
const { t } = useI18n()
const headingId = useId()
const store = useDesignStore()
const design = computed(() => store.design)
const preview = useAssetPreview(computed(() => props.link.slug), design)
const exporter = useAssetExport()
const toast = useToast()
const errorText = useErrorText()
// A failed download is a toast with a retry, so the editor never moves.
watch(() => exporter.error.value, (error) => {
  if (!error) return
  toast.error(errorText(error), {
    timeout: 8000,
    action: error.code === 'invalid_design' ? undefined : { label: t('generator.retry'), onClick: () => download(design.value.format) },
  })
})

const fullName = computed(() => props.link.currentFullName ?? props.link.slug)
const formats = computed(() => formatsFor(design.value))
const designUrl = computed(() => designLink(window.location.origin, fullName.value, design.value))
const { copied: designCopied, copy: copyDesign } = useCopy()

const pending = ref<Format | null>(null)

async function download(format: Format) {
  const next = { ...store.design, format }
  store.apply(next)
  const name = props.link.currentFullName?.split('/')[1] ?? props.link.slug
  pending.value = format
  try {
    await exporter.download(props.link.slug, next, assetFilename(name, format))
  }
  finally {
    pending.value = null
  }
}
</script>

<style scoped>
.design { display: grid; grid-template-columns: minmax(0, 480px) minmax(0, 1fr); gap: 32px; align-items: start; min-width: 0; }
.design__preview, .design__side { display: flex; flex-direction: column; gap: 16px; min-width: 0; }
.design__export { display: flex; flex-direction: column; gap: 8px; padding-top: 14px; border-top: 1px solid color-mix(in srgb, rgb(var(--v-theme-line)) 60%, transparent); }
.design__downloads { --s99ab-control-h: 40px; display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; }
.design__download { font-weight: 700; letter-spacing: 0.02em; }
.design__share { align-self: flex-start; margin-left: -8px; text-transform: none; letter-spacing: 0; font-weight: 600; }
@media (max-width: 760px) {
  .design { grid-template-columns: minmax(0, 1fr); }
  .design__preview { max-width: 360px; width: 100%; margin-inline: auto; }
}
</style>
