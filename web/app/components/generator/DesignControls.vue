<!-- app/components/generator/DesignControls.vue -->
<template>
  <div class="controls">
    <v-checkbox
      v-model="logo"
      :label="t('design.fields.logo')"
      hide-details
      density="compact"
      color="accent"
      class="controls__wide controls__logo"
    />
    <div class="controls__wide controls__colors">
      <QrControlField :label="t('design.fields.foreground')">
        <ColorField
          :id="`${id}-fg`"
          v-model="fg"
          :label="t('design.fields.foreground')"
        />
      </QrControlField>
      <v-btn
        icon="mdi-swap-horizontal"
        variant="text"
        size="small"
        class="controls__swap"
        :aria-label="t('design.swap')"
        :title="t('design.swap')"
        @click="swap"
      />
      <QrControlField :label="t('design.fields.background')">
        <ColorField
          :id="`${id}-bg`"
          v-model="bg"
          :label="t('design.fields.background')"
        />
      </QrControlField>
    </div>
    <QrControlField
      :label="t('design.fields.errorCorrection')"
      :hint="t(`design.ecHelper.${ec}`)"
      class="controls__wide"
    >
      <KeyGroup
        v-model="ec"
        :items="ecItems"
        :label="t('design.fields.errorCorrection')"
        choice
        class="keys--fill"
      />
    </QrControlField>
    <QrControlField :label="t('design.fields.size')">
      <NumberField
        :id="`${id}-s`"
        v-model="size"
        :label="t('design.fields.size')"
        suffix="px"
        :min="SIZE_RANGE.min"
        :max="SIZE_RANGE.max"
        :step="32"
      />
    </QrControlField>
    <QrControlField :label="t('design.fields.margin')">
      <NumberField
        :id="`${id}-m`"
        v-model="margin"
        :label="t('design.fields.margin')"
        :suffix="t('design.modules')"
        :min="4"
        :max="16"
        :step="1"
      />
    </QrControlField>
    <QrControlField
      v-if="!d.transparent"
      :label="t('design.fields.radius')"
    >
      <NumberField
        :id="`${id}-r`"
        v-model="radius"
        :label="t('design.fields.radius')"
        suffix="%"
        :min="0"
        :max="100"
        :step="5"
      />
    </QrControlField>
    <QrControlField :label="t('design.fields.transparent')">
      <KeyGroup
        v-model="transparentKey"
        :items="[{ value: 'no', label: t('design.opaque') }, { value: 'yes', label: t('design.transparent') }]"
        :label="t('design.fields.transparent')"
        choice
        class="keys--fill"
      />
    </QrControlField>
  </div>
</template>

<script setup lang="ts">
import KeyGroup from '~/components/common/KeyGroup.vue'
import { useDesignStore } from '~/stores/design'
import { EC_LEVELS, LOGO_EC_LEVELS, SIZE_RANGE, formatsFor, type Design } from '~/utils/design'
import ColorField from './ColorField.vue'
import NumberField from './NumberField.vue'
import QrControlField from './QrControlField.vue'

const { t } = useI18n()
const id = useId()
const store = useDesignStore()
const d = computed(() => store.design)

/** A two-way binding to one design field; other fields stay as they are. */
function field<K extends keyof Design>(key: K) {
  return computed<Design[K]>({
    get: () => store.design[key],
    set: (value) => {
      const next = { ...store.design, [key]: value }
      if (!formatsFor(next).includes(next.format)) next.format = 'png'
      // The logo needs a level that can recover the modules it replaces.
      if (next.logo && !LOGO_EC_LEVELS.includes(next.ec)) next.ec = 'H'
      store.apply(next)
    },
  })
}

const fg = field('fg')
const bg = field('bg')
const ec = field('ec')
const margin = field('margin')
const radius = field('radius')
const size = field('size')
const transparent = field('transparent')
const logo = field('logo')
function swap() {
  store.apply({ ...store.design, fg: store.design.bg, bg: store.design.fg })
}
const transparentKey = computed({
  get: () => (transparent.value ? 'yes' : 'no'),
  set: (v: string) => (transparent.value = v === 'yes'),
})
const ecItems = computed(() => EC_LEVELS.map(l => ({
  value: l,
  label: l,
  title: t(`design.ecHelper.${l}`),
  aria: `${l}, ${t(`design.ecHelper.${l}`)}`,
  disabled: d.value.logo && !LOGO_EC_LEVELS.includes(l),
})))
</script>

<style scoped>
.controls__logo { margin: 0; }
.controls__colors { display: grid; grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr); align-items: end; gap: 12px; }
.controls__swap { margin: 0 4px 2px; color: rgb(var(--v-theme-ink2)); }
.controls__logo :deep(.v-selection-control) { min-height: 32px; }
/* The font's capitals sit 2px above the line box centre; this lines them up with the box. */
.controls__logo :deep(.v-label) { position: relative; top: 2px; }
</style>
