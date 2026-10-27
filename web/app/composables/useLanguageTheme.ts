// app/composables/useLanguageTheme.ts
// While a result is open the site takes the colours of the repository's stack; without one it uses the visitor's.
import { Hct, argbFromHex, hexFromArgb } from '@material/material-color-utilities'
import { rebuildThemes, themeState } from '@smegg99/s99ab/theme'
import { useTheme } from 'vuetify'
import { LANGUAGE_COLORS } from '~/generated/languageColors'
import { BRAND } from '~/theme/brand'

type Accent = { light: string, dark: string }
type Themes = Parameters<typeof rebuildThemes>[0]

/** GitHub's colour for a language, or a neutral ink colour when it has none. */
export function languageColor(name: string): string {
  return LANGUAGE_COLORS[name] ?? 'rgb(var(--v-theme-ink3))'
}

/** The language colour's hue and chroma at the site's accent tones, so text on it keeps its contrast. */
export function languageAccent(language: string | undefined): Accent | undefined {
  const hex = language ? LANGUAGE_COLORS[language] : undefined
  if (!hex) return undefined
  const color = Hct.fromInt(argbFromHex(hex))
  const tone = (value: number) => hexFromArgb(Hct.from(color.hue, color.chroma, value).toInt())
  return { light: tone(BRAND.accentTone.light), dark: tone(BRAND.accentTone.dark) }
}

function asAccent(value: string | Accent | undefined, fallback: Accent): Accent {
  if (!value) return fallback
  return typeof value === 'string' ? { light: value, dark: value } : value
}

const accentVar = () => {
  const app = document.querySelector('.v-application')
  return app ? getComputedStyle(app).getPropertyValue('--v-theme-accent') : ''
}

/**
 * Repaints the themes once, inside a view transition: the browser cross-fades a snapshot of the old
 * colours into the live page, so nothing is recomputed per frame. Without the API, or with reduced
 * motion, the colours change at once.
 */
function fadeAccent(themes: Themes, to: Accent) {
  const doc = document as Document & { startViewTransition?: (update: () => Promise<void>) => { finished: Promise<void> } }
  if (!doc.startViewTransition || matchMedia('(prefers-reduced-motion: reduce)').matches) {
    rebuildThemes(themes, { accent: to })
    return
  }
  const root = document.documentElement
  root.classList.add('accent-fade')
  const transition = doc.startViewTransition(async () => {
    const before = accentVar()
    rebuildThemes(themes, { accent: to })
    // Vuetify writes its stylesheet asynchronously; the new snapshot waits for it, briefly.
    for (let i = 0; i < 20 && accentVar() === before; i++) await new Promise(resolve => setTimeout(resolve, 10))
  })
  transition.finished.finally(() => root.classList.remove('accent-fade'))
}

/** The largest language with a GitHub colour drives the accent; the page glow uses the first three. */
export function useLanguageTheme(languages: () => string[]) {
  const theme = useTheme()
  // The visitor's own accent, read before any repository colour replaces it.
  const own = asAccent(themeState().accent, { light: '#000000', dark: '#ffffff' })
  const colored = computed(() => languages().filter(name => LANGUAGE_COLORS[name]))
  watch(() => colored.value[0], (name, previous) => {
    if (name === previous) return
    fadeAccent(theme.themes.value, languageAccent(name) ?? own)
  })
  onBeforeUnmount(() => rebuildThemes(theme.themes.value, { accent: own }))
  return { glow: computed(() => colored.value.slice(0, 3).map(name => LANGUAGE_COLORS[name]!)) }
}
