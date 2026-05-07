<!-- app/components/home/ExportBar.vue -->
<template>
  <div class="d-flex flex-column ga-3 export-bar pt-4">
    <div class="d-flex justify-space-between align-end ga-4 flex-wrap">
      <div class="export-bar__corner d-flex flex-column ga-2">
        <div class="text-caption text-medium-emphasis">
          {{ t('home.qr.controls.cornerRadius') }}
        </div>
        <NumberField :id="`${id}-r`" v-model="cornerRadius" :aria-label="t('home.qr.controls.cornerRadius')"
          :ariaLabel="t('home.qr.controls.cornerRadius')" suffix="%" :min="0" :max="50" :step="1" />
      </div>

      <div class="export-bar__buttons d-flex flex-column ga-2">
        <div class="text-caption text-medium-emphasis">
          {{ t('home.qr.export.title') }}
        </div>
        <v-btn-group divided color="primary" variant="outlined" class="export-btn-group">
          <v-btn v-for="fmt in formats" :key="fmt" color="primary" variant="outlined" :disabled="busy" :loading="busy"
            class="text-none export-btn" @click="onExport(fmt)">
            {{ fmt.toUpperCase() }}
          </v-btn>
        </v-btn-group>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import NumberField from './NumberField.vue'

type Format = 'svg' | 'png' | 'jpg'

const props = defineProps<{ busy: boolean }>()
const emit = defineEmits<{
  (e: 'export', format: Format): void
}>()

const cornerRadius = defineModel<number>('cornerRadius', { required: true })

const { t } = useI18n()
const id = useId()
const formats: Format[] = ['svg', 'png', 'jpg']

void props

function onExport(format: Format) {
  emit('export', format)
}
</script>

<style scoped>
.export-bar {
  border-top: 1px dashed rgba(var(--v-theme-on-surface), 0.1);
}

.export-bar__corner {
  flex: 0 0 auto;
  width: 220px;
  max-width: 100%;
}

.export-bar__buttons {
  flex: 0 0 auto;
  min-width: 0;
}

.export-btn-group {
  height: 48px;
}

.export-btn {
  height: 48px !important;
  min-width: 80px;
  font-size: 0.95rem;
}
</style>
