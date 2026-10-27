<!-- app/pages/index.vue -->
<template>
  <div class="home-page site-width">
    <!-- The open repository's stack colours behind the page; outside the content so it fades with the backdrop. -->
    <Teleport to="#main">
      <Transition name="glow">
        <div
          v-if="glow.length"
          class="glow"
          :style="glowStyle"
          aria-hidden="true"
        />
      </Transition>
    </Teleport>
    <Transition
      name="swap"
      mode="out-in"
    >
      <GeneratorStart
        v-if="!flow.link.value"
        key="start"
        v-model="flow.rawUrl.value"
        :stage="flow.stage.value"
        :error="flow.error.value"
        :problems="flow.problems.value"
        @submit="flow.submit"
        @dismiss="flow.clearError"
      />
      <GeneratorResult
        v-else
        key="result"
        :link="flow.link.value"
        :outcome="flow.outcome.value"
        @close="flow.close"
        @refresh="flow.refresh"
      />
    </Transition>
    <p
      class="sr-only"
      role="status"
      aria-live="polite"
    >
      {{ flow.announcement.value ? t(flow.announcement.value) : '' }}
    </p>
  </div>
</template>

<script setup lang="ts">
import GeneratorResult from '~/components/generator/GeneratorResult.vue'
import GeneratorStart from '~/components/generator/GeneratorStart.vue'
import { useGeneratorFlow } from '~/composables/useGeneratorFlow'
import { useLanguageTheme } from '~/composables/useLanguageTheme'

const { t } = useI18n()
const flow = useGeneratorFlow()
const { glow } = useLanguageTheme(() => {
  const link = flow.link.value
  if (!link) return []
  return link.languages?.map(l => l.name) ?? (link.language ? [link.language] : [])
})
const SPOTS = ['12% 18%', '88% 26%', '50% 96%']
const glowStyle = computed(() => ({
  background: glow.value.map((color, i) => `radial-gradient(circle at ${SPOTS[i]}, ${color} 0, transparent 42%)`).join(', '),
}))
useSeoMeta({ title: () => t('home.metaTitle'), description: () => t('app.tagline') })
</script>

<style scoped>
.glow { position: fixed; inset: 0; z-index: -1; opacity: 0.22; pointer-events: none; }
.glow-enter-active, .glow-leave-active { transition: opacity 0.9s ease-in-out; }
.glow-enter-from, .glow-leave-to { opacity: 0; }
</style>
