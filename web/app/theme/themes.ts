// app/theme/themes.ts
import { s99abTheme } from '@smegg99/s99ab/theme'
import { BRAND } from './brand'

export const vuetifyOptions = s99abTheme({
  palette: BRAND.palette,
  chroma: BRAND.chroma,
  accent: BRAND.accent,
  defaultTheme: 'dark',
})
