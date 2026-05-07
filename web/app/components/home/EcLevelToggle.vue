<!-- app/components/home/EcLevelToggle.vue -->
<template>
  <v-select v-model="model" :aria-label="t('home.qr.controls.errorCorrection')" :items="items" density="comfortable"
    hide-details item-title="title" item-value="value" />
</template>

<script setup lang="ts">
import type { QrErrorLevel } from '~/composables/useQrCode'

const model = defineModel<QrErrorLevel>({ required: true })
const { t } = useI18n()

const ecLevels: QrErrorLevel[] = ['L', 'M', 'Q', 'H']

const items = computed(() => ecLevels.map(lvl => ({
  value: lvl,
  title: `${t(`home.qr.controls.ec.${lvl}`)} - ${t(`home.qr.controls.ecHelper.${lvl}`)}`,
})))
</script>
