<!-- app/components/home/HomeHero.vue -->
<template>
  <v-sheet class="text-center" color="transparent">
    <v-img src="/qr2g.it.svg" :alt="t('app.name')" :class="['hero-logo mx-auto mb-2', { 'hero-logo--invert': !isDark }]"
      :width="logoWidth" contain />
    <div role="heading" aria-level="1" class="text-h5 text-md-h4 font-weight-bold hero-heading">
      {{ t('home.hero.title') }}
      <div class="hero-heading__accent d-inline">
        {{ t('home.hero.titleAccent') }}
      </div>
    </div>
  </v-sheet>
</template>

<script setup lang="ts">
import { useDisplay } from 'vuetify'

const { t } = useI18n()
const settings = useSettingsStore()
const { mdAndUp } = useDisplay()

const isDark = computed(() => settings.resolvedTheme === 'dark')
const logoWidth = computed(() => mdAndUp.value ? 280 : 220)
</script>

<style scoped>
.hero-logo {
  max-width: 70vw;
}

.hero-logo--invert :deep(img) {
  filter: invert(1);
}

.hero-heading {
  letter-spacing: 0;
  line-height: 1.15;
}

.hero-heading__accent {
  background: linear-gradient(90deg, rgb(var(--v-theme-on-background)), rgb(var(--v-theme-primary)));
  background-clip: text;
  color: transparent;
  -webkit-background-clip: text;
}
</style>
