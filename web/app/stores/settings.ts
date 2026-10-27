// app/stores/settings.ts
import { defineStore } from 'pinia'
import {
  DEFAULT_THEME_MODE,
  type ThemeMode,
} from '~/types/config'
import { DEFAULT_LOCALE, LOCALE_CODES, type LocaleCode } from '~/types/locale'

/** The first supported browser language; a saved choice replaces it when the store hydrates. */
function browserLocale(): LocaleCode {
  if (typeof navigator === 'undefined') return DEFAULT_LOCALE
  for (const tag of navigator.languages ?? [navigator.language]) {
    const code = tag.toLowerCase().split('-')[0] as LocaleCode
    if (LOCALE_CODES.includes(code)) return code
  }
  return DEFAULT_LOCALE
}

interface SettingsState {
  themeMode: ThemeMode
  language: LocaleCode
}

export const useSettingsStore = defineStore('settings', {
  state: (): SettingsState => ({
    themeMode: DEFAULT_THEME_MODE,
    language: browserLocale(),
  }),
  actions: {
    setThemeMode(mode: ThemeMode) {
      this.themeMode = mode
    },
    setLanguage(code: LocaleCode) {
      this.language = code
    },
  },
  persist: {
    key: 'qr2g.settings',
    pick: ['themeMode', 'language'],
  },
})
