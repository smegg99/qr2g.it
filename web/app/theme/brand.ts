// app/theme/brand.ts
// Shared by the Vuetify theme and the pre-mount ground colour in nuxt.config.ts.
export const BRAND = {
  palette: 'graphite',
  accent: { light: '#0b7569', dark: '#3fc7b2' },
  // A faint teal cast on every neutral, so surfaces belong to the ground behind them.
  chroma: 0.12,
} as const
