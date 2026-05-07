<!-- app/components/home/ResultPanel.vue -->
<template>
  <v-sheet color="transparent" class="d-flex flex-column ga-4">
    <ResultLinkField :url="link.url" :label="t('home.result.shortLink')" :copy-label="t('home.result.copy')"
      :copied-label="t('home.result.copied')" :open-label="t('home.result.openShort')" />

    <ResultRepoTile :href="link.currentUrl" :title="repoName" :repo-label="t('home.result.repository')"
      :status-label="t('home.result.status')" :status-text="statusText" :state="stateKey" />
  </v-sheet>
</template>

<script setup lang="ts">
import type { LinkResponse } from '~/types/api'
import ResultLinkField from './ResultLinkField.vue'
import ResultRepoTile from './ResultRepoTile.vue'

const props = defineProps<{
  link: LinkResponse
}>()

const { t } = useI18n()

const stateKey = computed(() => {
  const s = props.link.state?.toLowerCase()
  if (s === 'active' || s === 'tombstoned' || s === 'stale') return s
  return 'unknown' as const
})

const statusText = computed(() => t(`home.result.states.${stateKey.value}`))
const repoName = computed(() => props.link.currentFullName || props.link.initialFullName)
</script>
