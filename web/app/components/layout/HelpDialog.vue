<!-- app/components/layout/HelpDialog.vue -->
<template>
  <v-dialog v-model="open" max-width="640" transition="fade-transition" content-class="dialog-glass" :scrollable="true">
    <v-card class="dialog-glass__inner pa-2 pa-sm-7 d-flex flex-column ga-5" rounded="lg">
      <div class="d-flex align-center justify-space-between ga-3">
        <div class="text-h6 font-weight-bold help-title">
          {{ t('help.title') }}
        </div>
        <v-btn icon="mdi-close" variant="text" density="comfortable" :aria-label="t('help.close')"
          @click="open = false" />
      </div>

      <v-divider :opacity="0.4" />

      <div class="text-body-1 text-medium-emphasis">
        {{ t('help.intro') }}
      </div>

      <HelpSection :title="t('help.howItWorks.title')" :items="[
        t('help.howItWorks.step1'),
        t('help.howItWorks.step2'),
        t('help.howItWorks.step3'),
      ]" ordered title-color="text-primary" />

      <HelpSection :title="t('help.qr.title')" :items="[t('help.qr.customize'), t('help.qr.export')]"
        title-color="text-secondary" />

      <HelpSection :title="t('help.privacy.title')" :items="[t('help.privacy.note')]" title-color="text-info" />
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import HelpSection from './HelpSection.vue'

const open = defineModel<boolean>({ required: true })
const { t } = useI18n()
</script>

<style scoped>
.help-title {
  background: linear-gradient(90deg, rgb(var(--v-theme-on-surface)), rgb(var(--v-theme-primary)));
  background-clip: text;
  color: transparent;
  -webkit-background-clip: text;
}
</style>
