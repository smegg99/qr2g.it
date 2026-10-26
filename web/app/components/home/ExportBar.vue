<!-- app/components/home/ExportBar.vue -->
<template>
  <KeyGroup
    :items="items"
    :label="t('home.qr.export.title')"
    :disabled="busy"
    @press="value => emit('export', value as Format)"
  />
</template>

<script setup lang="ts">
import KeyGroup from '~/components/common/KeyGroup.vue'

type Format = 'svg' | 'png' | 'jpg'
const props = defineProps<{ busy: boolean }>()
const emit = defineEmits<{ (e: 'export', format: Format): void }>()
const { t } = useI18n()
const formats: Format[] = ['svg', 'png', 'jpg']
const items = computed(() => formats.map(format => ({
  value: format,
  label: format.toUpperCase(),
  title: `${t('home.qr.export.title')} ${format.toUpperCase()}`,
  aria: props.busy ? t('home.qr.export.downloading') : `${t('home.qr.export.title')} ${format.toUpperCase()}`,
})))
</script>
