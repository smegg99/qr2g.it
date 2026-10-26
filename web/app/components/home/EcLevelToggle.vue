<!-- app/components/home/EcLevelToggle.vue -->
<template>
  <KeyGroup
    v-model="level"
    :items="items"
    :label="t('home.qr.controls.errorCorrection')"
    choice
    class="keys--fill"
  />
</template>

<script setup lang="ts">
import KeyGroup from '~/components/common/KeyGroup.vue'
import type { QrErrorLevel } from '~/composables/useQrCode'

const model = defineModel<QrErrorLevel>({ required: true })
const { t } = useI18n()

const ecLevels: QrErrorLevel[] = ['L', 'M', 'Q', 'H']
const level = computed({
  get: () => model.value,
  set: v => (model.value = v as QrErrorLevel),
})

const items = computed(() => ecLevels.map(lvl => ({
  value: lvl,
  label: t(`home.qr.controls.ec.${lvl}`),
  title: t(`home.qr.controls.ecHelper.${lvl}`),
  aria: `${t(`home.qr.controls.ec.${lvl}`)}, ${t(`home.qr.controls.ecHelper.${lvl}`)}`,
})))
</script>
