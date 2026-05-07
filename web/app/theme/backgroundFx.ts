import { normalizeHexColor, type MaterialAccentColors } from './materialAccent'

interface BackgroundFxOptions {
  accentHex: string
  background: string
  colors: MaterialAccentColors
  isDark: boolean
}

const sharedBgConfig = {
  speed: 3.2,
  horizontalPressure: 3,
  verticalPressure: 4,
  waveFrequencyX: 2,
  waveFrequencyY: 3,
  waveAmplitude: 4,
  shadows: 4,
  highlights: 3,
  wireframe: false,
  colorBlending: 7,
  backgroundAlpha: 1,
  grainScale: 2,
  grainSparsity: 0,
  grainIntensity: 0.08,
  grainSpeed: 1.2,
  resolution: 0.9,
  yOffset: 0,
  yOffsetWaveMultiplier: 2.4,
  yOffsetColorMultiplier: 2.7,
  yOffsetFlowMultiplier: 3,
  flowDistortionA: 1.6,
  flowDistortionB: 2.6,
  flowScale: 1.5,
  flowEase: 0.42,
  flowEnabled: true,
  enableProceduralTexture: false,
  textureVoidLikelihood: 0.06,
  textureVoidWidthMin: 10,
  textureVoidWidthMax: 500,
  textureBandDensity: 0.8,
  textureColorBlending: 0.06,
  textureSeed: 333,
  textureEase: 0.68,
  textureShapeTriangles: 20,
  textureShapeCircles: 15,
  textureShapeBars: 15,
  textureShapeSquiggles: 10,
  domainWarpEnabled: false,
  domainWarpIntensity: 0,
  domainWarpScale: 3,
  vignetteIntensity: 0.15,
  vignetteRadius: 0.84,
  fresnelEnabled: false,
  fresnelPower: 2,
  fresnelIntensity: 0.5,
  fresnelColor: '#FFFFFF',
  iridescenceEnabled: false,
  iridescenceIntensity: 0.5,
  iridescenceSpeed: 1,
  bloomIntensity: 0.2,
  bloomThreshold: 0.64,
  chromaticAberration: 0,
}

export function createBackgroundFxConfig(options: BackgroundFxOptions) {
  const seed = hashString(`${options.accentHex}-${options.isDark ? 'dark' : 'light'}`)
  const background = normalizeHexColor(options.background)

  return {
    ...sharedBgConfig,
    speed: 2.8 + seedRatio(seed, 0) * 0.9,
    waveFrequencyX: 1.8 + seedRatio(seed, 8) * 2.3,
    waveFrequencyY: 2.1 + seedRatio(seed, 16) * 2.2,
    waveAmplitude: options.isDark ? 3.6 + seedRatio(seed, 24) : 2.8 + seedRatio(seed, 24),
    colorBlending: options.isDark ? 6.6 : 5.4,
    colorBrightness: options.isDark ? 1.14 : 1.03,
    colorSaturation: options.isDark ? 3.1 : 1.55,
    flowDistortionA: 1.1 + seedRatio(seed, 4) * 1.4,
    flowDistortionB: 1.8 + seedRatio(seed, 12) * 1.4,
    textureSeed: seed % 10000,
    grainIntensity: options.isDark ? 0.08 : 0.04,
    vignetteIntensity: options.isDark ? 0.15 : 0.06,
    fresnelColor: options.colors.accent,
    backgroundColor: background,
    proceduralBackgroundColor: background,
    colors: createColorStops(options.colors, background, options.isDark, seed),
  }
}

function createColorStops(
  colors: MaterialAccentColors,
  background: string,
  isDark: boolean,
  seed: number,
) {
  const source = [
    colors.primary,
    colors.secondary,
    colors.accent,
    colors['primary-soft'],
    colors['secondary-soft'],
    colors['accent-soft'],
  ]
  const rotated = rotate(source, seed % source.length)
  const weights = isDark
    ? [0.46, 0.38, 0.42, 0.24, 0.28, 0.25]
    : [0.24, 0.2, 0.22, 0.34, 0.3, 0.32]

  return rotated.map((color, index) => ({
    color: mixHex(background, color, weights[index] ?? 0.3),
    enabled: true,
    influence: 0.12 + index * 0.03,
  }))
}

function rotate<T>(items: T[], offset: number): T[] {
  return [...items.slice(offset), ...items.slice(0, offset)]
}

function mixHex(baseHex: string, tintHex: string, amount: number): string {
  const base = hexToRgb(baseHex)
  const tint = hexToRgb(tintHex)
  const mixed = base.map((channel, index) =>
    Math.round(channel + ((tint[index] ?? channel) - channel) * amount),
  )

  return `#${mixed.map(value => value.toString(16).padStart(2, '0')).join('')}`
}

function hexToRgb(value: string): [number, number, number] {
  const hex = normalizeHexColor(value).slice(1)
  return [
    Number.parseInt(hex.slice(0, 2), 16),
    Number.parseInt(hex.slice(2, 4), 16),
    Number.parseInt(hex.slice(4, 6), 16),
  ]
}

function hashString(value: string): number {
  let hash = 2166136261
  for (let i = 0; i < value.length; i++) {
    hash ^= value.charCodeAt(i)
    hash = Math.imul(hash, 16777619)
  }
  return hash >>> 0
}

function seedRatio(seed: number, shift: number): number {
  return ((seed >>> shift) & 0xFF) / 255
}
