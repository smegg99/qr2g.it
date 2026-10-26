<!-- app/components/home/HomeResolved.vue -->
<template>
  <Pane>
    <PaneHead
      :title="t('home.result.title')"
      tag="h1"
    >
      <template #actions>
        <v-btn
          icon="mdi-close"
          variant="text"
          size="small"
          :aria-label="t('home.result.close')"
          :title="t('home.result.close')"
          @click="emit('close')"
        />
      </template>
    </PaneHead>
    <ResultPanel :link="link" />
    <QrSection
      :text="link.url"
      :filename-hint="filenameHint"
    />
  </Pane>
</template>

<script setup lang="ts">
import type { LinkResponse } from '~/types/api'
import Pane from '~/components/common/Pane.vue'
import PaneHead from '~/components/common/PaneHead.vue'
import QrSection from './QrSection.vue'
import ResultPanel from './ResultPanel.vue'

const props = defineProps<{
  link: LinkResponse
}>()

const emit = defineEmits<{ (e: 'close'): void }>()

const { t } = useI18n()

const filenameHint = computed(() => props.link.url.replace(/^https?:\/\//i, '').replace(/\/$/, ''))
</script>
