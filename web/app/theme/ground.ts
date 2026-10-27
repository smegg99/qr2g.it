// app/theme/ground.ts
import type { NeatConfig } from '@firecms/neat'

type Stop = readonly [color: string, influence: number]
type Shape = Pick<NeatConfig, 'waveFrequencyX' | 'waveFrequencyY'>

interface Scene {
  name: string
  dark: readonly Stop[]
  light: readonly Stop[]
  shape: Shape
}

export const SCENES: readonly Scene[] = [
  {
    name: 'tide',
    dark: [['#163a3a', 0.22], ['#1b2847', 0.2], ['#1b4039', 0.18], ['#352f3c', 0.1], ['#101c28', 0.3]],
    light: [['#e0f2ee', 0.22], ['#e3e9f6', 0.2], ['#d7f0e7', 0.18], ['#f2e9df', 0.1], ['#ebeff7', 0.3]],
    shape: { waveFrequencyX: 1.4, waveFrequencyY: 1.8 },
  },
  {
    name: 'aurora',
    dark: [['#1a3d31', 0.22], ['#21403d', 0.2], ['#2c2947', 0.18], ['#3d2c42', 0.1], ['#131d2b', 0.3]],
    light: [['#e0f1e8', 0.22], ['#d8f0ea', 0.2], ['#e7e2f3', 0.18], ['#f1e5ee', 0.1], ['#edf1f4', 0.3]],
    shape: { waveFrequencyX: 1.8, waveFrequencyY: 1.3 },
  },
  {
    name: 'ember',
    dark: [['#342838', 0.22], ['#3e2d2d', 0.2], ['#403628', 0.18], ['#213c36', 0.1], ['#201a27', 0.3]],
    light: [['#f2e6e0', 0.22], ['#f2e0db', 0.2], ['#f0e5f1', 0.18], ['#deefe7', 0.1], ['#f3eeeb', 0.3]],
    shape: { waveFrequencyX: 1.2, waveFrequencyY: 1.5 },
  },
  {
    name: 'glacier',
    dark: [['#1e3143', 0.22], ['#233b4d', 0.2], ['#234244', 0.18], ['#344250', 0.1], ['#121d2e', 0.3]],
    light: [['#e0eef7', 0.22], ['#d7ebf4', 0.2], ['#dcf2ed', 0.18], ['#ebf1f7', 0.1], ['#e7f0f7', 0.3]],
    shape: { waveFrequencyX: 1.6, waveFrequencyY: 1.7 },
  },
  {
    name: 'moss',
    dark: [['#283828', 0.22], ['#353f27', 0.2], ['#234035', 0.18], ['#403a28', 0.1], ['#19251e', 0.3]],
    light: [['#e7f1de', 0.22], ['#e3efd8', 0.2], ['#dcf0e6', 0.18], ['#f2ecd9', 0.1], ['#edf3e7', 0.3]],
    shape: { waveFrequencyX: 1.7, waveFrequencyY: 1.4 },
  },
  {
    name: 'dusk',
    dark: [['#292643', 0.22], ['#3c2a44', 0.2], ['#432f3c', 0.18], ['#233e42', 0.1], ['#1b192f', 0.3]],
    light: [['#e6e2f4', 0.22], ['#f1e2f1', 0.2], ['#f2e4e1', 0.18], ['#deefed', 0.1], ['#edecf4', 0.3]],
    shape: { waveFrequencyX: 1.3, waveFrequencyY: 1.9 },
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
    // Scene selection still works without local storage.
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
  return {
    colors: stops.map(([color, influence]) => ({ color, influence, enabled: true })),
    speed: 0.55,
    horizontalPressure: 2.6,
    verticalPressure: 2.9,
    waveAmplitude: 1.7,
    shadows: 1,
    highlights: 1,
    colorBlending: 9,
    colorSaturation: 0,
    colorBrightness: 1,
    grainScale: 2,
    grainIntensity: dark ? 0.02 : 0.015,
    grainSparsity: 0,
    grainSpeed: 0,
    vignetteIntensity: dark ? 0.08 : 0.03,
    vignetteRadius: 0.85,
    flowEnabled: false,
    domainWarpEnabled: false,
    wireframe: false,
    backgroundColor: background,
    backgroundAlpha: 1,
    // The shader is blurred by design, so half resolution looks the same at a quarter of the cost.
    resolution: 0.5,
    yOffset: 0,
    ...scene.shape,
    ...(still ? { speed: 0, grainSpeed: 0 } : {}),
  }
}
