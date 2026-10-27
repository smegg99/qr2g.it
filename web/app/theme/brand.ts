// app/theme/brand.ts
import { Hct, hexFromArgb } from '@material/material-color-utilities'

export const BRAND = {
  palette: 'graphite',
  chroma: 0.65,
  accentChroma: 48,
  accentTone: { light: 40, dark: 80 },
} as const

export function visitorAccent(seed: string): { light: string, dark: string } {
  const hue = Number.parseInt(seed, 16) / 0x100000000 * 360
  const tone = (value: number) => hexFromArgb(Hct.from(hue, BRAND.accentChroma, value).toInt())
  return { light: tone(BRAND.accentTone.light), dark: tone(BRAND.accentTone.dark) }
}
