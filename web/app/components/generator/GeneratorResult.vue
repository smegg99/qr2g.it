<!-- app/components/generator/GeneratorResult.vue -->
<template>
  <Pane class="result-pane">
    <PaneHead
      :title="t('result.title')"
      tag="h1"
    >
      <template #actions>
        <v-btn
          icon
          variant="text"
          size="small"
          :aria-busy="refreshing || undefined"
          :aria-label="t('result.refresh')"
          :title="t('result.refresh')"
          @click="refresh"
        >
          <v-icon
            icon="mdi-refresh"
            class="result-refresh"
            :style="{ transform: `rotate(${turns * 360}deg)` }"
          />
        </v-btn>
        <ResultSteps v-model="step" />
        <v-btn
          icon="mdi-close"
          variant="text"
          size="small"
          :aria-label="t('result.close')"
          :title="t('result.close')"
          @click="emit('close')"
        />
      </template>
    </PaneHead>
    <Transition
      :name="step === 'qr' ? 'step-forward' : 'step-back'"
      mode="out-in"
    >
      <div
        v-if="step === 'link'"
        key="link"
        class="pane-body result-step"
      >
        <LinkCard
          :link="link"
          :outcome="outcome"
        />
        <div class="result-step__nav">
          <v-btn
            color="accent"
            variant="flat"
            append-icon="mdi-arrow-right"
            class="result-step__go"
            :text="t('result.toQr')"
            @click="step = 'qr'"
          />
        </div>
      </div>
      <div
        v-else
        key="qr"
        class="pane-body result-step"
      >
        <DesignSection :link="link" />
      </div>
    </Transition>
  </Pane>
</template>

<script setup lang="ts">
import { useQueryCache } from '@pinia/colada'
import Pane from '~/components/common/Pane.vue'
import { getLinkStatsQueryKey } from '~/generated/api/@pinia/colada.gen'
import PaneHead from '~/components/common/PaneHead.vue'
import type { Link } from '~/generated/api/types.gen'
import type { Outcome } from '~/composables/useGenerator'
import DesignSection from './DesignSection.vue'
import LinkCard from './LinkCard.vue'
import ResultSteps, { type ResultStep } from './ResultSteps.vue'

const props = defineProps<{ link: Link, outcome: Outcome | null }>()
const emit = defineEmits<{ (e: 'close' | 'refresh'): void }>()
const { t } = useI18n()
const route = useRoute()
const router = useRouter()

// The step lives in the URL hash, so the browser's back button returns from the QR code to the link.
const step = computed<ResultStep>({
  get: () => (route.hash === '#qr' ? 'qr' : 'link'),
  set: next => void router.push({ hash: next === 'qr' ? '#qr' : '' }),
})
// Refresh reads the link and its clicks again; the counters roll to the new numbers. The icon turns
// once per click and keeps turning while the work runs, so even an instant answer is seen.
const TURN_MS = 600
const cache = useQueryCache()
const refreshing = ref(false)
const turns = ref(0)
async function refresh() {
  if (refreshing.value) return
  refreshing.value = true
  turns.value++
  const spin = setInterval(() => turns.value++, TURN_MS)
  emit('refresh')
  try {
    await Promise.all([
      cache.invalidateQueries({ key: getLinkStatsQueryKey({ path: { slug: props.link.slug } }) }),
      new Promise(resolve => setTimeout(resolve, TURN_MS)),
    ])
  }
  finally {
    clearInterval(spin)
    refreshing.value = false
  }
}

onBeforeUnmount(() => {
  if (route.hash) void router.replace({ hash: '' })
})
</script>

<style scoped>
.result-step { display: flex; flex-direction: column; gap: 16px; }
/* Both steps keep one size on a desktop screen, so switching never resizes the pane. */
@media (min-width: 960px) and (min-height: 700px) {
  .result-step { height: 560px; }
  /* The link card takes the step's height; its chart grows into the room. */
  .result-step > :deep(.display) { flex: 1 1 auto; min-height: 0; grid-template-rows: auto 1fr auto; }
}
.result-step__nav { display: flex; justify-content: flex-end; margin-top: auto; }
.result-step__go { text-transform: none; letter-spacing: 0; font-weight: 700; }
/* Forward slides in from the right, back from the left: a quick fade with a short travel. */
.step-forward-leave-active, .step-back-leave-active { transition: opacity 0.1s ease-in, transform 0.1s ease-in; }
.step-forward-enter-active, .step-back-enter-active { transition: opacity 0.18s ease-out, transform 0.18s cubic-bezier(0.2, 0.8, 0.2, 1); }
.step-forward-enter-from, .step-back-leave-to { opacity: 0; transform: translateX(10px); }
.step-forward-leave-to, .step-back-enter-from { opacity: 0; transform: translateX(-10px); }
.result-refresh { transition: transform 0.6s cubic-bezier(0.4, 0, 0.2, 1); }
@media (max-width: 600px) {
  /* The steps need the head's width on a phone; the title stays for assistive technology. */
  .result-pane :deep(.pane-head .pane-title) { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0, 0, 0, 0); }
  .result-pane :deep(.pane-actions) { flex: 1 1 auto; justify-content: space-between; }
}
@media (prefers-reduced-motion: reduce) {
  .result-refresh { transition: none; }
  .step-forward-enter-active, .step-forward-leave-active, .step-back-enter-active, .step-back-leave-active { transition: opacity 0.15s ease; transform: none; }
}
</style>
