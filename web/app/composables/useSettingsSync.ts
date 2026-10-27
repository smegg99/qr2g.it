// app/composables/useSettingsSync.ts
import { useLocale } from 'vuetify'
import { useAppearance } from '@smegg99/s99ab/appearance'
import { useSettingsStore } from '~/stores/settings'

export function useSettingsSync() {
  const settings = useSettingsStore()
  const appearance = useAppearance()
  const vuetifyLocale = useLocale()
  const { locale, setLocale } = useI18n()

  watch(() => settings.themeMode, (mode) => {
    appearance.setMode(mode === 'auto' ? 'system' : mode, undefined, 'none')
  }, { immediate: true })

  watch(() => settings.language, (code) => {
    vuetifyLocale.current.value = code
    if (locale.value !== code) void setLocale(code)
  }, { immediate: true })

  return { settings }
}
