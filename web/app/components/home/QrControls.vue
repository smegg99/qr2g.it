<!-- app/components/home/QrControls.vue -->
<template>
  <div class="controls">
    <QrControlField :label="t('home.qr.controls.foreground')">
      <ColorField
        :id="`${id}-fg`"
        v-model="foreground"
        :label="t('home.qr.controls.foreground')"
      />
    </QrControlField>
    <QrControlField :label="t('home.qr.controls.background')">
      <ColorField
        :id="`${id}-bg`"
        v-model="background"
        :label="t('home.qr.controls.background')"
      />
    </QrControlField>
    <QrControlField
      :label="t('home.qr.controls.errorCorrection')"
      :hint="t(`home.qr.controls.ecHelper.${errorCorrection}`)"
      class="controls__wide"
    >
      <EcLevelToggle v-model="errorCorrection" />
    </QrControlField>
    <QrControlField :label="t('home.qr.controls.size')">
      <NumberField
        :id="`${id}-size`"
        v-model="size"
        :label="t('home.qr.controls.size')"
        :suffix="t('home.qr.controls.px')"
        :min="64"
        :max="4096"
        :step="32"
      />
    </QrControlField>
    <QrControlField :label="t('home.qr.controls.margin')">
      <NumberField
        :id="`${id}-m`"
        v-model="margin"
        :label="t('home.qr.controls.margin')"
        :suffix="t('home.qr.controls.modules')"
        :min="0"
        :max="16"
        :step="1"
      />
    </QrControlField>
    <QrControlField :label="t('home.qr.controls.cornerRadius')">
      <NumberField
        :id="`${id}-r`"
        v-model="cornerRadius"
        :label="t('home.qr.controls.cornerRadius')"
        suffix="%"
        :min="0"
        :max="50"
        :step="1"
      />
    </QrControlField>
  </div>
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
const cornerRadius = defineModel<number>('cornerRadius', { required: true })
const errorCorrection = defineModel<QrErrorLevel>('errorCorrection', { required: true })

const { t } = useI18n()
const id = useId()
</script>
