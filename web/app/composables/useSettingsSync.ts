// app/composables/useSettingsSync.ts
import { useLocale, useTheme } from 'vuetify'
import { useSettingsStore } from '~/stores/settings'
import { THEME_MODE, type ThemeMode } from '~/types/config'
import type { LocaleCode } from '~/types/locale'

export function useSettingsSync() {
  const settings = useSettingsStore()
  const vuetifyTheme = useTheme()
  const vuetifyLocale = useLocale()
  const { locale, setLocale } = useI18n()

  function applyTheme(mode: ThemeMode) {
    if (mode === THEME_MODE.AUTO) {
      const prefersDark = typeof window !== 'undefined'
        && window.matchMedia?.('(prefers-color-scheme: dark)').matches
      vuetifyTheme.change(prefersDark ? 'dark' : 'light')
    }
    else {
      vuetifyTheme.change(mode)
    }
    if (import.meta.client) {
      const name = vuetifyTheme.global.name.value
      document.documentElement.style.colorScheme = name
      const background = vuetifyTheme.themes.value[name]?.colors.background
      if (typeof background === 'string') document.documentElement.style.backgroundColor = background
    }
  }

  function applyLocale(code: LocaleCode) {
    vuetifyLocale.current.value = code
    if (locale.value !== code) setLocale(code)
  }

  watch(() => settings.themeMode, applyTheme, { immediate: true })
  watch(() => settings.language, applyLocale, { immediate: true })

  let mql: MediaQueryList | null = null
  const onMqlChange = () => {
    if (settings.themeMode === THEME_MODE.AUTO) applyTheme(THEME_MODE.AUTO)
  }

  onMounted(() => {
    if (typeof window !== 'undefined' && window.matchMedia) {
      mql = window.matchMedia('(prefers-color-scheme: dark)')
      mql.addEventListener('change', onMqlChange)
    }
  })

  onBeforeUnmount(() => {
    if (mql) mql.removeEventListener('change', onMqlChange)
  })

  return {
    settings,
  }
}
