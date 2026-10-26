<!-- app/app.vue -->
<template>
  <div v-cloak>
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
  </div>
</template>

<script setup lang="ts">
import { SITE } from '~/content/site'

const settings = useSettingsStore()
const route = useRoute()
useHead(() => ({
  htmlAttrs: { lang: settings.language },
  link: [{ rel: 'canonical', href: SITE.origin + route.path }],
}))
</script>
