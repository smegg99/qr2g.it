// app/theme/visitor.ts
export const ACCENT_KEY = 'qr2g.accent'
export const PAINT_KEY = 'qr2g.paint'

export function visitorSeed(): string {
  try {
    const saved = localStorage.getItem(ACCENT_KEY)
    if (saved && /^[0-9a-f]{8}$/i.test(saved)) return saved
  }
  catch {
    // A blocked store still gets one accent for this visit.
  }

  const seed = crypto.getRandomValues(new Uint32Array(1))[0]!.toString(16).padStart(8, '0')
  try {
    localStorage.setItem(ACCENT_KEY, seed)
  }
  catch {
    // The current visit can use the seed without persisting it.
  }
  return seed
}
