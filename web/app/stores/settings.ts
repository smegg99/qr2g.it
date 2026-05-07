// app/stores/settings.ts
import { defineStore } from 'pinia'
import {
  DEFAULT_THEME_MODE,
  THEME_MODE,
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
  getters: {
    isDark(state): boolean {
      if (state.themeMode === THEME_MODE.DARK) return true
      if (state.themeMode === THEME_MODE.LIGHT) return false
      if (typeof window === 'undefined') return true
      return window.matchMedia?.('(prefers-color-scheme: dark)').matches ?? true
    },
    resolvedTheme(): 'dark' | 'light' {
      return this.isDark ? 'dark' : 'light'
    },
  },
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
