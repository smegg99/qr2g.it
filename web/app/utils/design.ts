// app/utils/design.ts
// One QR design model for the preview, every export and the prefilled design link.
// Names and bounds follow the asset operation in contracts/backend/paths/assets.yaml.

export const FORMATS = ['svg', 'png', 'jpg'] as const
export type Format = typeof FORMATS[number]
export const EC_LEVELS = ['L', 'M', 'Q', 'H'] as const
export type EcLevel = typeof EC_LEVELS[number]
/** Error correction levels that can recover the modules the centre logo replaces. */
export const LOGO_EC_LEVELS: readonly EcLevel[] = ['H']

export interface Design {
  format: Format
  fg: string
  bg: string
  ec: EcLevel
  margin: number
  radius: number
  size: number
  transparent: boolean
  logo: boolean
}

const COLOR = /^#[0-9a-f]{6}$/i
// eslint-disable-next-line no-control-regex
const CONTROL = /[\u0000-\u001f\u007f]/

const RANGES = {
  margin: { min: 0, max: 16 },
  radius: { min: 0, max: 100 },
  size: { min: 128, max: 4000 },
} as const
export const SIZE_RANGE = RANGES.size

export const DEFAULT_DESIGN: Readonly<Design> = Object.freeze({
  format: 'png',
  fg: '#111111',
  bg: '#ffffff',
  ec: 'H',
  margin: 4,
  radius: 0,
  size: 1024,
  transparent: false,
  logo: true,
})

/** Which formats a design can export; JPG cannot carry transparency. */
export function formatsFor(d: Design): Format[] {
  return d.transparent ? ['svg', 'png'] : [...FORMATS]
}

/** The asset operation's query parameters; server defaults are left out to keep URLs short. */
export function assetQuery(d: Design, format: Format = d.format): Record<string, string> {
  const q: Record<string, string> = { format }
  const set = (key: string, value: string | number | boolean, fallback: string | number | boolean) => {
    if (value !== fallback) q[key] = String(value)
  }
  set('fg', d.fg, DEFAULT_DESIGN.fg)
  set('bg', d.bg, DEFAULT_DESIGN.bg)
  set('logo', d.logo, true)
  set('ec', d.ec, d.logo ? 'H' : 'M')
  set('margin', d.margin, DEFAULT_DESIGN.margin)
  set('size', d.size, DEFAULT_DESIGN.size)
  if (!d.transparent) set('radius', d.radius, DEFAULT_DESIGN.radius)
  set('transparent', d.transparent, false)
  return q
}

/** A problem found while importing a prefilled design link. */
export interface DesignProblem {
  param: string
  reason: 'unknown' | 'invalid' | 'out_of_range' | 'repository'
}

export interface ImportedDesign {
  repo: string
  design: Design
  problems: DesignProblem[]
}

const PARAM_KEYS = ['format', 'fg', 'bg', 'ec', 'margin', 'radius', 'size', 'transparent', 'logo'] as const
type ParamKey = typeof PARAM_KEYS[number]

/** Applies one validated parameter; returns a problem reason or null. */
function applyParam(d: Design, key: ParamKey, raw: string): DesignProblem['reason'] | null {
  switch (key) {
    case 'format':
    case 'ec': {
      const allowed: readonly string[] = key === 'ec' ? EC_LEVELS : FORMATS
      const value = key === 'ec' ? raw.toUpperCase() : raw
      if (!allowed.includes(value)) return 'invalid'
      Object.assign(d, { [key]: value })
      return null
    }
    case 'fg':
    case 'bg':
      if (!COLOR.test(raw)) return 'invalid'
      d[key] = raw.toLowerCase()
      return null
    case 'transparent':
    case 'logo':
      if (raw !== 'true' && raw !== 'false') return 'invalid'
      d[key] = raw === 'true'
      return null
    default: {
      if (!/^\d{1,5}$/.test(raw)) return 'invalid'
      const n = Number(raw)
      const range = RANGES[key]
      if (n < range.min || n > range.max) return 'out_of_range'
      d[key] = n
      return null
    }
  }
}

/**
 * Reads a prefilled design link. Unknown or invalid parameters are reported and
 * ignored; nothing here fetches, redirects, renders HTML or touches permissions.
 */
export function importDesign(params: URLSearchParams, base: Design = DEFAULT_DESIGN): ImportedDesign {
  const design: Design = { ...base }
  const problems: DesignProblem[] = []
  let repo = ''
  for (const [key, raw] of params) {
    if (key === 'repo') {
      if (raw.length > 512 || CONTROL.test(raw)) problems.push({ param: key, reason: 'repository' })
      else repo = raw.trim()
      continue
    }
    if (!(PARAM_KEYS as readonly string[]).includes(key)) {
      problems.push({ param: key.slice(0, 40), reason: 'unknown' })
      continue
    }
    const reason = applyParam(design, key as ParamKey, raw)
    if (reason) problems.push({ param: key, reason })
  }
  if (design.logo && !LOGO_EC_LEVELS.includes(design.ec)) {
    problems.push({ param: 'ec', reason: 'invalid' })
    design.ec = 'H'
  }
  if (!formatsFor(design).includes(design.format)) design.format = 'png'
  return { repo, design, problems }
}

/** Rebuilds a stored design through the same validation as a design link. */
export function designFromObject(raw: unknown): Design {
  if (!raw || typeof raw !== 'object') return { ...DEFAULT_DESIGN }
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(raw as Record<string, unknown>)) {
    if ((PARAM_KEYS as readonly string[]).includes(key)
      && (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean')) {
      params.set(key, String(value))
    }
  }
  return importDesign(params).design
}

/** The canonical prefilled link: repository plus every non-default design choice. */
export function designLink(origin: string, repo: string, d: Design): string {
  const params = new URLSearchParams()
  if (repo) params.set('repo', repo)
  for (const key of PARAM_KEYS) {
    if (d[key] !== DEFAULT_DESIGN[key]) params.set(key, String(d[key]))
  }
  return `${origin.replace(/\/$/, '')}/?${params.toString()}`
}

/** A download name such as Goptivum-qr.png, matching the server's Content-Disposition. */
export function assetFilename(name: string, format: Format): string {
  const base = name.replace(/[^A-Za-z0-9._-]+/g, '_').replace(/^[._]+|[._]+$/g, '') || 'qr2git'
  return `${base}-qr.${format}`
}
