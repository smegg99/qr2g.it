import {
  argbFromHex,
  hexFromArgb,
  themeFromSourceColor,
} from '@material/material-color-utilities'

export const FALLBACK_ACCENT_HEX = '#7C9BFF'

export interface MaterialAccentColors {
  'primary': string
  'on-primary': string
  'secondary': string
  'on-secondary': string
  'accent': string
  'on-accent': string
  'error': string
  'on-error': string
  'primary-soft': string
  'primary-alt': string
  'secondary-soft': string
  'secondary-alt': string
  'accent-soft': string
  'accent-alt': string
}

export function getDailyAccentHex(date = new Date()): string {
  const key = `${date.getFullYear()}-${date.getMonth() + 1}-${date.getDate()}`
  const seed = hashString(key)
  const hue = seed % 360
  const saturation = 68 + ((seed >>> 8) % 18)
  const lightness = 48 + ((seed >>> 16) % 10)

  return hslToHex(hue, saturation, lightness)
}

export function buildMaterialAccentColors(sourceHex: string, isDark: boolean): MaterialAccentColors {
  const sourceArgb = argbFromHex(normalizeHexColor(sourceHex))
  const materialTheme = themeFromSourceColor(sourceArgb)
  const scheme = isDark ? materialTheme.schemes.dark : materialTheme.schemes.light

  return {
    'primary': hexFromArgb(scheme.primary),
    'on-primary': hexFromArgb(scheme.onPrimary),
    'secondary': hexFromArgb(scheme.secondary),
    'on-secondary': hexFromArgb(scheme.onSecondary),
    'accent': hexFromArgb(scheme.tertiary),
    'on-accent': hexFromArgb(scheme.onTertiary),
    'error': hexFromArgb(scheme.error),
    'on-error': hexFromArgb(scheme.onError),
    'primary-soft': hexFromArgb(scheme.primaryContainer),
    'primary-alt': hexFromArgb(scheme.onPrimaryContainer),
    'secondary-soft': hexFromArgb(scheme.secondaryContainer),
    'secondary-alt': hexFromArgb(scheme.onSecondaryContainer),
    'accent-soft': hexFromArgb(scheme.tertiaryContainer),
    'accent-alt': hexFromArgb(scheme.onTertiaryContainer),
  }
}

export function normalizeHexColor(value: string): string {
  const trimmed = value.trim()
  if (/^#[0-9a-f]{6}$/i.test(trimmed)) return trimmed
  if (/^#[0-9a-f]{3}$/i.test(trimmed)) {
    const [, r, g, b] = trimmed
    return `#${r}${r}${g}${g}${b}${b}`
  }
  return FALLBACK_ACCENT_HEX
}

function hashString(value: string): number {
  let hash = 2166136261
  for (let i = 0; i < value.length; i++) {
    hash ^= value.charCodeAt(i)
    hash = Math.imul(hash, 16777619)
  }
  return hash >>> 0
}

function hslToHex(hue: number, saturation: number, lightness: number): string {
  const s = saturation / 100
  const l = lightness / 100
  const c = (1 - Math.abs(2 * l - 1)) * s
  const x = c * (1 - Math.abs((hue / 60) % 2 - 1))
  const m = l - c / 2
  const [r, g, b] = hueToRgb(hue, c, x).map(channel =>
    Math.round((channel + m) * 255),
  ) as [number, number, number]

  return `#${toHex(r)}${toHex(g)}${toHex(b)}`
}

function hueToRgb(hue: number, c: number, x: number): [number, number, number] {
  if (hue < 60) return [c, x, 0]
  if (hue < 120) return [x, c, 0]
  if (hue < 180) return [0, c, x]
  if (hue < 240) return [0, x, c]
  if (hue < 300) return [x, 0, c]
  return [c, 0, x]
}

function toHex(value: number): string {
  return value.toString(16).padStart(2, '0')
}
