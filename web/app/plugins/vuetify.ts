// app/plugins/vuetify.ts
import '@mdi/font/css/materialdesignicons.css'
import 'vuetify/styles'
import { createVuetify } from 'vuetify'
import { en, pl } from 'vuetify/locale'
import { s99abTheme } from '@smegg99/s99ab/theme'
import { BRAND, visitorAccent } from '~/theme/brand'
import { PAINT_KEY, visitorSeed } from '~/theme/visitor'

export default defineNuxtPlugin((app) => {
  const saved = useCookie<{ themeMode?: string }>('qr2g.settings').value?.themeMode
  const dark = saved === 'dark' || (saved !== 'light' && import.meta.client && matchMedia('(prefers-color-scheme: dark)').matches)
  const seed = import.meta.client ? visitorSeed() : '00000000'
  const vuetify = createVuetify({
    ...s99abTheme({
      palette: BRAND.palette,
      chroma: BRAND.chroma,
      accent: visitorAccent(seed),
      defaultTheme: dark ? 'dark' : 'light',
    }),
    locale: { locale: 'en', fallback: 'en', messages: { en, pl } },
  })
  app.vueApp.use(vuetify)

  if (import.meta.client) {
    const theme = vuetify.theme
    watch(() => [theme.current.value.dark, theme.current.value.colors.background], () => {
      const current = theme.current.value
      document.documentElement.style.colorScheme = current.dark ? 'dark' : 'light'
      document.documentElement.style.backgroundColor = String(current.colors.background)
      try {
        localStorage.setItem(PAINT_KEY, JSON.stringify({
          seed,
          light: theme.themes.value.light?.colors.background,
          dark: theme.themes.value.dark?.colors.background,
        }))
      }
      catch {
        // Pre-mount paint caching is optional when storage is blocked.
      }
    }, { immediate: true })
  }
})
