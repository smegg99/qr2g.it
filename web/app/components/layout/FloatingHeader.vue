<!-- app/components/layout/FloatingHeader.vue -->
<template>
  <div class="floating-header">
    <v-btn icon="mdi-translate" variant="text" size="small" density="comfortable" :aria-label="t('header.language')"
      @click="cycleLocale" />
    <v-btn :icon="themeIcon" variant="text" size="small" density="comfortable" :aria-label="t('header.theme')"
      @click="cycleTheme" />
    <v-btn icon="mdi-help-circle-outline" variant="text" size="small" density="comfortable"
      :aria-label="t('header.help')" @click="helpOpen = true" />
    <v-btn icon="mdi-github" variant="text" size="small" density="comfortable" href="https://github.com/smegg99/qr2g.it"
      target="_blank" rel="noopener" :aria-label="t('header.repository')" />
  </div>

  <HelpDialog v-model="helpOpen" />
</template>

<script setup lang="ts">
import { useSettingsStore } from '~/stores/settings'
import { THEME_MODES, type ThemeMode } from '~/types/config'
import { LOCALE_CODES } from '~/types/locale'
import HelpDialog from './HelpDialog.vue'

const { t } = useI18n()
const settings = useSettingsStore()

const helpOpen = ref(false)

const themeMode = computed<ThemeMode>(() => settings.themeMode)

const themeIcon = computed(() => {
  switch (themeMode.value) {
    case 'auto': return 'mdi-theme-light-dark'
    case 'light': return 'mdi-white-balance-sunny'
    case 'dark': return 'mdi-moon-waning-crescent'
  }
  return 'mdi-theme-light-dark'
})

function cycleTheme() {
  const idx = THEME_MODES.indexOf(themeMode.value)
  const next = THEME_MODES[(idx + 1) % THEME_MODES.length] ?? 'auto'
  settings.setThemeMode(next)
}

function cycleLocale() {
  const idx = LOCALE_CODES.indexOf(settings.language)
  const next = LOCALE_CODES[(idx + 1) % LOCALE_CODES.length] ?? LOCALE_CODES[0]
  if (next) settings.setLanguage(next)
}
</script>

<style scoped>
.floating-header {
  position: fixed;
  top: 16px;
  right: 16px;
  z-index: 1000;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px;
  border-radius: 999px;
  background: rgba(var(--v-theme-surface), 0.55);
  backdrop-filter: blur(14px) saturate(140%);
  -webkit-backdrop-filter: blur(14px) saturate(140%);
  border: thin solid rgba(var(--v-theme-on-surface), 0.06);
}
</style>
