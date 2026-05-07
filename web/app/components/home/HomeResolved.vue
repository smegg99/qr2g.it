<!-- app/components/home/HomeResolved.vue -->
<template>
  <GlassCard class="pa-5 pa-sm-7 d-flex flex-column ga-5">
    <div class="d-flex align-center justify-space-between ga-3">
      <div class="text-h6 font-weight-bold result-title">
        {{ t('home.result.title') }}
      </div>
      <v-btn icon="mdi-close" variant="text" size="small" density="comfortable" :aria-label="t('home.result.close')"
        @click="emit('close')" />
    </div>

    <ResultPanel :link="link" />
    <v-divider :opacity="0.5" />
    <QrSection :text="link.url" :filename-hint="filenameHint" />
  </GlassCard>
</template>

<script setup lang="ts">
import type { LinkResponse } from '~/types/api'
import GlassCard from '~/components/common/GlassCard.vue'
import QrSection from './QrSection.vue'
import ResultPanel from './ResultPanel.vue'

const props = defineProps<{
  link: LinkResponse
}>()

const emit = defineEmits<{ (e: 'close'): void }>()

const { t } = useI18n()

const filenameHint = computed(() => {
  const url = props.link.url
  return url.replace(/^https?:\/\//i, '').replace(/\/$/, '')
})
</script>

<style scoped>
.result-title {
  background: linear-gradient(90deg, rgb(var(--v-theme-on-surface)), rgb(var(--v-theme-primary)));
  background-clip: text;
  color: transparent;
  -webkit-background-clip: text;
}
</style>
