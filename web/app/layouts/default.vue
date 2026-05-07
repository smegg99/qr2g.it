<!-- app/layouts/default.vue -->
<template>
  <v-app>
    <ClientOnly>
      <BackgroundFx />
    </ClientOnly>
    <FloatingHeader />
    <v-main>
      <slot />
    </v-main>
    <SplashScreen />
  </v-app>
</template>

<script setup lang="ts">
import BackgroundFx from '~/components/layout/BackgroundFx.vue'
import FloatingHeader from '~/components/layout/FloatingHeader.vue'
import SplashScreen from '~/components/layout/SplashScreen.vue'
import { useSettingsSync } from '~/composables/useSettingsSync'

useSettingsSync()
</script>
