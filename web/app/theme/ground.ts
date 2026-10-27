// app/theme/ground.ts
import type { NeatConfig } from '@firecms/neat'

type Stop = readonly [color: string, influence: number]
type Shape = Partial<Omit<NeatConfig, 'colors' | 'ref'>>

interface Scene {
  name: string
  dark: readonly Stop[]
  light: readonly Stop[]
  shape: Shape
  darkShape?: Shape
}

export const SCENES: readonly Scene[] = [
  {
    name: 'tide',
    dark: [['#0b4f4a', 0.2], ['#26307e', 0.22], ['#0f7a6c', 0.16], ['#5c3a12', 0.06], ['#151a3d', 0.24]],
    light: [['#a6e6da', 0.22], ['#c3c9f6', 0.22], ['#7fd0c3', 0.16], ['#efe3c6', 0.05], ['#dfe4f7', 0.24]],
    shape: { waveFrequencyX: 2.2, waveFrequencyY: 3.1, waveAmplitude: 3.2, horizontalPressure: 3, verticalPressure: 4 },
    darkShape: { waveAmplitude: 4.2 },
  },
  {
    name: 'aurora',
    dark: [['#073d2c', 0.2], ['#11a37f', 0.14], ['#3b2a8a', 0.2], ['#86307a', 0.08], ['#08141c', 0.3]],
    light: [['#b3f0da', 0.2], ['#86dcc4', 0.16], ['#d2c8f7', 0.2], ['#f3c8e8', 0.1], ['#e8f4f2', 0.28]],
    shape: {
      waveFrequencyX: 4.2, waveFrequencyY: 1.2, waveAmplitude: 5, horizontalPressure: 1.6, verticalPressure: 6,
      flowEnabled: true, flowDistortionA: 3.4, flowDistortionB: 2.2, flowScale: 2.4, flowEase: 0.4,
    },
  },
  {
    name: 'ember',
    dark: [['#2c1033', 0.24], ['#6e2a2a', 0.2], ['#a3561d', 0.14], ['#0d4a46', 0.05], ['#160b1a', 0.28]],
    light: [['#f2d5c6', 0.22], ['#efc2ae', 0.16], ['#e6d0f2', 0.2], ['#b8e6dc', 0.1], ['#f7ede6', 0.26]],
    shape: { waveFrequencyX: 1.2, waveFrequencyY: 1.6, waveAmplitude: 6.5, horizontalPressure: 4, verticalPressure: 3, colorBlending: 9, speed: 1.8 },
  },
  {
    name: 'glacier',
    dark: [['#0b2f4a', 0.24], ['#1d6390', 0.18], ['#0f7f88', 0.16], ['#b9dcea', 0.04], ['#071420', 0.3]],
    light: [['#cfe8f7', 0.24], ['#a6d6ea', 0.18], ['#bde8e3', 0.16], ['#ffffff', 0.12], ['#e3eef6', 0.26]],
    shape: {
      waveFrequencyX: 2.8, waveFrequencyY: 2.4, waveAmplitude: 2.6, highlights: 7, speed: 1.6,
      domainWarpEnabled: true, domainWarpIntensity: 0.45, domainWarpScale: 2.6,
    },
    darkShape: { highlights: 4 },
  },
  {
    name: 'moss',
    dark: [['#1d3a1a', 0.24], ['#3f5c1e', 0.16], ['#0f6e5c', 0.18], ['#6e5c18', 0.08], ['#0c1710', 0.3]],
    light: [['#d6ecc2', 0.22], ['#c3e1ab', 0.16], ['#ade0d1', 0.18], ['#f1e5b2', 0.1], ['#eef4e6', 0.26]],
    shape: {
      waveFrequencyX: 3.2, waveFrequencyY: 2, waveAmplitude: 3.6, horizontalPressure: 5, verticalPressure: 2.5,
      flowEnabled: true, flowDistortionA: 1.2, flowDistortionB: 3.2, flowScale: 1.4, flowEase: 0.6,
    },
  },
  {
    name: 'dusk',
    dark: [['#28205e', 0.22], ['#5c2a6c', 0.16], ['#a04a5c', 0.1], ['#0f6a70', 0.12], ['#110e2a', 0.3]],
    light: [['#d5cff6', 0.22], ['#ecc8e9', 0.16], ['#f6d2c8', 0.12], ['#bfe6e2', 0.14], ['#efecf8', 0.26]],
    shape: { waveFrequencyX: 1.4, waveFrequencyY: 4.6, waveAmplitude: 4.4, horizontalPressure: 7, verticalPressure: 1.6, speed: 2 },
  },
]

const LAST_KEY = 'qr2g.ground'

// Avoid repeating the previous scene across page loads.
export function pickScene(): Scene {
  let last = ''
  try {
    last = localStorage.getItem(LAST_KEY) ?? ''
  }
  catch {
    // Storage can be blocked; the draw then only lacks the repeat guard.
  }
  const pool = SCENES.filter(scene => scene.name !== last)
  const scene = pool[Math.floor(Math.random() * pool.length)] ?? SCENES[0]!
  try {
    localStorage.setItem(LAST_KEY, scene.name)
  }
  catch {
    // Scene selection still works without session storage.
  }
  return scene
}

// The CSS washes drawn under the canvas, shown when WebGL is missing or still loading.
export function sceneWashes(scene: Scene, dark: boolean): string[] {
  const stops = dark ? scene.dark : scene.light
  return [stops[0]![0], stops[2]![0], stops[3]![0]]
}

export function groundConfig(scene: Scene, dark: boolean, background: string, still: boolean): Omit<NeatConfig, 'ref'> {
  const stops = dark ? scene.dark : scene.light
  const shape = { ...scene.shape, ...(dark ? scene.darkShape : undefined) }
  return {
    colors: stops.map(([color, influence]) => ({ color, influence, enabled: true })),
    speed: 2.4,
    horizontalPressure: 3,
    verticalPressure: 4,
    waveFrequencyX: 2.2,
    waveFrequencyY: 3.1,
    waveAmplitude: 3.6,
    shadows: dark ? 5 : 2,
    highlights: dark ? 3 : 5,
    colorBlending: dark ? 7 : 6,
    colorSaturation: dark ? 1.6 : 1.1,
    colorBrightness: dark ? 1 : 1.04,
    grainScale: 2,
    grainIntensity: dark ? 0.07 : 0.04,
    grainSparsity: 0,
    grainSpeed: 1,
    vignetteIntensity: dark ? 0.18 : 0.06,
    vignetteRadius: 0.85,
    wireframe: false,
    backgroundColor: background,
    backgroundAlpha: 1,
    // The shader is blurred by design, so half resolution looks the same at a quarter of the cost.
    resolution: 0.5,
    yOffset: 0,
    ...shape,
    ...(still ? { speed: 0, grainSpeed: 0 } : {}),
  }
}
