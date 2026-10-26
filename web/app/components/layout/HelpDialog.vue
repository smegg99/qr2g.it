<!-- app/components/layout/HelpDialog.vue -->
<template>
  <v-dialog
    v-model="open"
    max-width="600"
    scrollable
    :aria-labelledby="headingId"
  >
    <Pane class="help-pane">
      <PaneHead
        :id="headingId"
        :title="t('help.title')"
      >
        <template #meta>
          <span class="help-version">v{{ config.public.version }}</span>
        </template>
        <template #actions>
          <v-btn
            icon="mdi-close"
            variant="text"
            size="small"
            :aria-label="t('help.close')"
            :title="t('help.close')"
            @click="open = false"
          />
        </template>
      </PaneHead>
      <div class="pane-body help-body">
        <p class="help-intro">
          {{ t('help.intro') }}
        </p>
        <HelpSection
          :title="t('help.howItWorks.title')"
          :items="[
            t('help.howItWorks.step1'),
            t('help.howItWorks.step2'),
            t('help.howItWorks.step3'),
          ]"
          ordered
        />
        <HelpSection
          :title="t('help.qr.title')"
          :items="[t('help.qr.customize'), t('help.qr.export')]"
        />
        <HelpSection
          :title="t('help.privacy.title')"
          :items="[t('help.privacy.note')]"
        />
      </div>
    </Pane>
  </v-dialog>
</template>

<script setup lang="ts">
import Pane from '~/components/common/Pane.vue'
import PaneHead from '~/components/common/PaneHead.vue'
import HelpSection from './HelpSection.vue'

const open = defineModel<boolean>({ required: true })
const { t } = useI18n()
const headingId = useId()
const config = useRuntimeConfig()
</script>
