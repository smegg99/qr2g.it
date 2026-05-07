<!-- app/components/home/QrControls.vue -->
<template>
  <v-sheet color="transparent" class="d-flex flex-column ga-5">
    <v-row no-gutters class="control-row">
      <v-col cols="12" sm="6">
        <QrControlField :label="t('home.qr.controls.foreground')">
          <ColorField :id="`${id}-fg`" v-model="foreground" :label="t('home.qr.controls.foreground')" />
        </QrControlField>
      </v-col>
      <v-col cols="12" sm="6">
        <QrControlField :label="t('home.qr.controls.background')">
          <ColorField :id="`${id}-bg`" v-model="background" :label="t('home.qr.controls.background')" />
        </QrControlField>
      </v-col>
    </v-row>

    <QrControlField :label="t('home.qr.controls.errorCorrection')">
      <EcLevelToggle v-model="errorCorrection" />
    </QrControlField>

    <v-row no-gutters class="control-row">
      <v-col cols="12" sm="6">
        <QrControlField :label="t('home.qr.controls.size')">
          <NumberField :id="`${id}-size`" v-model="size" :aria-label="t('home.qr.controls.size')"
            :ariaLabel="t('home.qr.controls.size')" :suffix="t('home.qr.controls.px')" :min="64" :max="4096"
            :step="32" />
        </QrControlField>
      </v-col>
      <v-col cols="12" sm="6">
        <QrControlField :label="t('home.qr.controls.margin')">
          <NumberField :id="`${id}-m`" v-model="margin" :aria-label="t('home.qr.controls.margin')"
            :ariaLabel="t('home.qr.controls.margin')" :min="0" :max="16" :step="1" />
        </QrControlField>
      </v-col>
    </v-row>
  </v-sheet>
</template>

<script setup lang="ts">
import type { QrErrorLevel } from '~/composables/useQrCode'
import ColorField from './ColorField.vue'
import EcLevelToggle from './EcLevelToggle.vue'
import NumberField from './NumberField.vue'
import QrControlField from './QrControlField.vue'

const foreground = defineModel<string>('foreground', { required: true })
const background = defineModel<string>('background', { required: true })
const size = defineModel<number>('size', { required: true })
const margin = defineModel<number>('margin', { required: true })
const errorCorrection = defineModel<QrErrorLevel>('errorCorrection', { required: true })

const { t } = useI18n()
const id = useId()
</script>

<style scoped>
.control-row {
  gap: 16px 0;
}

@media (min-width: 600px) {
  .control-row>.v-col:first-child {
    padding-right: 12px;
  }

  .control-row>.v-col:last-child {
    padding-left: 12px;
  }
}
</style>
