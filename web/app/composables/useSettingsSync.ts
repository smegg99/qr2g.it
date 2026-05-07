// app/composables/useSettingsSync.ts
import { useTheme } from 'vuetify'
import { useDailyAccentHex, useMaterialAccentTheme } from '~/composables/useMaterialAccentTheme'
import { useSettingsStore } from '~/stores/settings'
import { THEME_MODE, type ThemeMode } from '~/types/config'
import type { LocaleCode } from '~/types/locale'

export function useSettingsSync() {
  const settings = useSettingsStore()
  const vuetifyTheme = useTheme()
  const { locale, setLocale } = useI18n()
  const accentHex = useDailyAccentHex()
  const accentTheme = useMaterialAccentTheme(accentHex)

  function applyTheme(mode: ThemeMode) {
    if (mode === THEME_MODE.AUTO) {
      const prefersDark = typeof window !== 'undefined'
        && window.matchMedia?.('(prefers-color-scheme: dark)').matches
      vuetifyTheme.global.name.value = prefersDark ? 'dark' : 'light'
    }
    else {
      vuetifyTheme.global.name.value = mode
    }
  }

  function applyLocale(code: LocaleCode) {
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
    accentHex,
    accentColors: accentTheme.colors,
  }
}
