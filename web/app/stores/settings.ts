// app/stores/settings.ts
import { defineStore } from 'pinia'
import {
  DEFAULT_THEME_MODE,
  type ThemeMode,
} from '~/types/config'
import { DEFAULT_LOCALE, type LocaleCode } from '~/types/locale'

interface SettingsState {
  themeMode: ThemeMode
  language: LocaleCode
}

export const useSettingsStore = defineStore('settings', {
  state: (): SettingsState => ({
    themeMode: DEFAULT_THEME_MODE,
    language: DEFAULT_LOCALE,
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
