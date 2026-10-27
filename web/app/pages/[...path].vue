<!-- app/pages/[...path].vue -->
<!-- The server sends browsers here for unknown, unavailable or removed links. -->
<template>
  <div class="sub-page sub-page--narrow site-width">
    <Pane>
      <div class="pane-body">
        <UiEmptyState
          :title="title"
          :text="text"
          :icon="state === 'missing' ? 'mdi-link-off' : 'mdi-eye-off-outline'"
        >
          <template #actions>
            <v-btn
              color="accent"
              variant="flat"
              to="/"
              :text="t('missing.home')"
            />
          </template>
        </UiEmptyState>
      </div>
    </Pane>
  </div>
</template>

<script setup lang="ts">
import Pane from '~/components/common/Pane.vue'
import { getLink } from '~/generated/api/sdk.gen'

const SLUG = /^[1-9A-Z][0-9A-Z]{0,11}$/
const { t } = useI18n()
const route = useRoute()
const state = ref<'missing' | 'unavailable' | 'removed' | 'checking'>('checking')

const segment = computed(() => {
  const path = route.params.path
  return Array.isArray(path) && path.length === 1 ? String(path[0]) : ''
})

onMounted(async () => {
  if (!SLUG.test(segment.value)) {
    state.value = 'missing'
    return
  }
  const { data } = await getLink({ path: { slug: segment.value } })
  if (!data) state.value = 'missing'
  else state.value = data.state === 'removed' ? 'removed' : data.state === 'active' ? 'missing' : 'unavailable'
})

const title = computed(() => t(`missing.${state.value}.title`))
const text = computed(() => t(`missing.${state.value}.text`))
useSeoMeta({ title: () => `${title.value} - qr2g.it`, robots: 'noindex' })
</script>
