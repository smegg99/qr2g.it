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
    dark: [['#1c3434', 0.22], ['#222b40', 0.2], ['#213a35', 0.18], ['#35313a', 0.1], ['#141c24', 0.3]],
    light: [['#e3efec', 0.22], ['#e6eaf3', 0.2], ['#dbece6', 0.18], ['#efe9e2', 0.1], ['#edf0f5', 0.3]],
    shape: { waveFrequencyX: 1.4, waveFrequencyY: 1.8 },
  },
  {
    name: 'aurora',
    dark: [['#20372f', 0.22], ['#263b39', 0.2], ['#302e42', 0.18], ['#3b303e', 0.1], ['#171e27', 0.3]],
    light: [['#e3eee8', 0.22], ['#dcece8', 0.2], ['#e8e5f0', 0.18], ['#efe7ed', 0.1], ['#eef1f3', 0.3]],
    shape: { waveFrequencyX: 1.8, waveFrequencyY: 1.3 },
  },
  {
    name: 'ember',
    dark: [['#332b35', 0.22], ['#3b3030', 0.2], ['#3c352c', 0.18], ['#253733', 0.1], ['#201c25', 0.3]],
    light: [['#efe7e3', 0.22], ['#eee2df', 0.2], ['#eee7ef', 0.18], ['#e1ece7', 0.1], ['#f2eeec', 0.3]],
    shape: { waveFrequencyX: 1.2, waveFrequencyY: 1.5 },
  },
  {
    name: 'glacier',
    dark: [['#24313d', 0.22], ['#2a3a46', 0.2], ['#283d3e', 0.18], ['#39424b', 0.1], ['#171e29', 0.3]],
    light: [['#e4edf3', 0.22], ['#dce9ef', 0.2], ['#e0eeeb', 0.18], ['#edf1f5', 0.1], ['#eaf0f4', 0.3]],
    shape: { waveFrequencyX: 1.6, waveFrequencyY: 1.7 },
  },
  {
    name: 'moss',
    dark: [['#2b352b', 0.22], ['#343b2b', 0.2], ['#283b34', 0.18], ['#3c382c', 0.1], ['#1b231e', 0.3]],
    light: [['#e7eee1', 0.22], ['#e3ebdc', 0.2], ['#dfede6', 0.18], ['#eeeadd', 0.1], ['#edf1e9', 0.3]],
    shape: { waveFrequencyX: 1.7, waveFrequencyY: 1.4 },
  },
  {
    name: 'dusk',
    dark: [['#2d2b3e', 0.22], ['#3a2e40', 0.2], ['#40323b', 0.18], ['#283a3d', 0.1], ['#1e1d2b', 0.3]],
    light: [['#e8e5f1', 0.22], ['#eee4ee', 0.2], ['#efe6e4', 0.18], ['#e1eceb', 0.1], ['#eeedf3', 0.3]],
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
    colorSaturation: -0.25,
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
