// app/utils/time.ts

const UNITS: [Intl.RelativeTimeFormatUnit, number][] = [['day', 86_400], ['hour', 3_600], ['minute', 60]]

/** "5 minutes ago" in the given language, or `now` for anything under a minute. */
export function timeAgo(iso: string, locale: string, now: string): string {
  const seconds = (Date.parse(iso) - Date.now()) / 1000
  const format = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' })
  for (const [unit, size] of UNITS) {
    if (Math.abs(seconds) >= size) return format.format(Math.round(seconds / size), unit)
  }
  return now
}
