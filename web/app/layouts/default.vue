<!-- app/layouts/default.vue -->
<template>
  <v-app>
    <UiBackdrop />
    <a
      class="skip-link"
      href="#main"
    >{{ t('header.skip') }}</a>
    <SiteHeader />
    <ScrollRail />
    <UiToastHost />
    <v-main
      id="main"
      tabindex="-1"
    >
      <div class="page-body">
        <slot />
      </div>
    </v-main>
  </v-app>
</template>

<script setup lang="ts">
import ScrollRail from '~/components/layout/ScrollRail.vue'
import SiteHeader from '~/components/layout/SiteHeader.vue'
import { useSettingsSync } from '~/composables/useSettingsSync'

const { t } = useI18n()
useSettingsSync()
</script>
