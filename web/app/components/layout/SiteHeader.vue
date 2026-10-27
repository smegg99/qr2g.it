<!-- app/components/layout/SiteHeader.vue -->
<template>
  <header class="site-header">
    <NuxtLink
      v-if="route.path !== '/'"
      to="/"
      class="header-brand"
      :aria-label="t('header.home')"
    >
      <img
        src="/qr2g.it.svg"
        alt=""
        width="100"
        height="33"
      >
    </NuxtLink>
    <div class="header-keys">
      <nav
        class="header-links"
        :aria-label="t('header.links')"
      >
        <v-btn
          to="/docs"
          icon="mdi-console-line"
          variant="text"
          :aria-label="t('header.docs')"
          :title="t('header.docs')"
        />
        <v-btn
          icon="mdi-email-outline"
          variant="text"
          :aria-label="t('header.contact')"
          :title="t('header.contact')"
          @click="contactOpen = true"
        />
      </nav>
      <span
        class="header-sep"
        aria-hidden="true"
      />
      <v-btn
        variant="text"
        class="header-lang"
        :aria-label="t('header.language')"
        :title="t('header.language')"
        @click="cycleLocale"
      >
        {{ settings.language === 'en' ? 'EN' : 'PL' }}
      </v-btn>
      <v-btn
        :icon="themeIcon"
        variant="text"
        :aria-label="themeLabel"
        :title="themeLabel"
        @click="cycleTheme"
      />
      <v-btn
        icon="mdi-help-circle-outline"
        variant="text"
        :aria-label="t('header.help')"
        :title="t('header.help')"
        @click="helpOpen = true"
      />
      <span
        class="header-sep"
        aria-hidden="true"
      />
      <v-btn
        icon="mdi-github"
        variant="text"
        :href="SITE.repository"
        target="_blank"
        rel="noopener noreferrer"
        :aria-label="t('header.repository')"
        :title="t('header.repository')"
      />
      <v-btn
        icon="mdi-linkedin"
        variant="text"
        :href="SITE.author"
        target="_blank"
        rel="noopener noreferrer"
        aria-label="LinkedIn"
        title="LinkedIn"
      />
    </div>
  </header>
  <HelpDialog v-model="helpOpen" />
  <ContactDialog v-model="contactOpen" />
</template>

<script setup lang="ts">
import { SITE } from '~/content/site'
import { useSettingsStore } from '~/stores/settings'
import { THEME_MODES } from '~/types/config'
import { LOCALE_CODES } from '~/types/locale'
import ContactDialog from './ContactDialog.vue'
import HelpDialog from './HelpDialog.vue'

const { t } = useI18n()
const route = useRoute()
const settings = useSettingsStore()
const helpOpen = ref(false)
const contactOpen = ref(false)
const themeIcon = computed(() => ({ auto: 'mdi-theme-light-dark', light: 'mdi-white-balance-sunny', dark: 'mdi-moon-waning-crescent' })[settings.themeMode])
const themeLabel = computed(() => `${t('header.theme')}: ${t(`settings.theme.${settings.themeMode}`)}`)
function cycleTheme() {
  settings.setThemeMode(THEME_MODES[(THEME_MODES.indexOf(settings.themeMode) + 1) % THEME_MODES.length] ?? 'auto')
}
function cycleLocale() {
  settings.setLanguage(LOCALE_CODES[(LOCALE_CODES.indexOf(settings.language) + 1) % LOCALE_CODES.length] ?? 'en')
}
</script>
