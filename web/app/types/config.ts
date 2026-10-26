// app/types/config.ts
export const THEME_MODE = {
  AUTO: 'auto',
  LIGHT: 'light',
  DARK: 'dark',
} as const

export type ThemeMode = typeof THEME_MODE[keyof typeof THEME_MODE]

export const DEFAULT_THEME_MODE: ThemeMode = THEME_MODE.AUTO

export const THEME_MODES = Object.values(THEME_MODE) as ThemeMode[]
