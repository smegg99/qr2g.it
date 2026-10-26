// app/composables/useTurnstile.ts

interface TurnstileWindow {
  turnstile?: {
    render: (el: HTMLElement | string, opts: TurnstileRenderOptions) => string
    reset: (id: string) => void
    remove: (id: string) => void
    execute: (id: string) => void
  }
}

interface TurnstileRenderOptions {
  'sitekey': string
  'size'?: 'normal' | 'compact' | 'flexible'
  'execution'?: 'render' | 'execute'
  'appearance'?: 'always' | 'execute' | 'interaction-only'
  'action'?: string
  'retry'?: 'auto' | 'never'
  'retry-interval'?: number
  'refresh-expired'?: 'auto' | 'manual' | 'never'
  'callback'?: (token: string) => void
  'error-callback'?: (err: unknown) => boolean
  'expired-callback'?: () => void
  'timeout-callback'?: () => void
  'unsupported-callback'?: () => void
}

const SCRIPT_URL = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
const CONTAINER_ID = 'qr2g-turnstile-host'
const ACTION = 'create_link'

let scriptPromise: Promise<void> | null = null
let widgetId: string | null = null
let pendingResolve: ((token: string) => void) | null = null
let pendingReject: ((err: Error) => void) | null = null

function loadScript(): Promise<void> {
  if (typeof window === 'undefined') return Promise.reject(new Error('turnstile: window not available'))
  const tw = window as unknown as TurnstileWindow
  if (tw.turnstile) return Promise.resolve()
  if (scriptPromise) return scriptPromise

  const script = document.querySelector<HTMLScriptElement>(`script[src^="${SCRIPT_URL.split('?')[0]}"]`)
    ?? document.createElement('script')
  scriptPromise = new Promise<void>((resolve, reject) => {
    script.addEventListener('load', () => resolve(), { once: true })
    script.addEventListener('error', () => reject(new Error('turnstile: script load failed')), { once: true })
    if (!script.src) {
      script.src = SCRIPT_URL
      script.async = true
      script.defer = true
      document.head.appendChild(script)
    }
  }).catch((err: unknown) => {
    script.remove()
    scriptPromise = null
    throw err
  })
  return scriptPromise
}

function ensureContainer(): HTMLElement {
  const el = document.getElementById(CONTAINER_ID)
  if (!el) throw new Error('turnstile: container not mounted')
  return el
}

function removeWidget() {
  if (widgetId) {
    (window as unknown as TurnstileWindow).turnstile?.remove(widgetId)
    widgetId = null
  }
}

function settleResolve(token: string) {
  pendingResolve?.(token)
  pendingResolve = null
  pendingReject = null
  removeWidget()
}

function settleReject(err: Error) {
  pendingReject?.(err)
  pendingResolve = null
  pendingReject = null
  removeWidget()
}

export function useTurnstile() {
  const config = useRuntimeConfig()
  const siteKey = (config.public.turnstileSiteKey as string) || ''
  const isConfigured = !!siteKey

  async function getToken(): Promise<string | null> {
    if (!isConfigured) return null
    if (typeof window === 'undefined') return null

    for (let attempt = 0; attempt < 2; attempt++) {
      try {
        await loadScript()
        break
      }
      catch (err) {
        if (attempt === 1) throw err
      }
    }
    const turnstile = (window as unknown as TurnstileWindow).turnstile
    if (!turnstile) throw new Error('turnstile: API unavailable')

    return new Promise<string>((resolve, reject) => {
      pendingReject?.(new Error('turnstile: superseded by a new request'))
      removeWidget()
      pendingResolve = resolve
      pendingReject = reject

      const onCallback = (token: string) => settleResolve(token)
      let failures = 0
      const onError = (err: unknown) => {
        const code = String(err)
        const retryable = /^(300|600)/.test(code) || ['200500', '110600', '110620'].includes(code)
        const terminal = !retryable || ++failures >= 3
        if (terminal) settleReject(new Error(`turnstile: ${code}`))
        return !terminal
      }

      try {
        widgetId = turnstile.render(ensureContainer(), {
          'sitekey': siteKey,
          'execution': 'execute',
          'appearance': 'execute',
          'action': ACTION,
          'retry': 'auto',
          'retry-interval': 1000,
          'refresh-expired': 'auto',
          'callback': onCallback,
          'error-callback': onError,
          'expired-callback': () => {
            if (!widgetId) return
            try {
              turnstile.reset(widgetId)
            }
            catch {
              // ignore: widget already gone
            }
          },
          'timeout-callback': () => settleReject(new Error('turnstile: challenge timed out')),
          'unsupported-callback': () => settleReject(new Error('turnstile: browser unsupported')),
        })
        turnstile.execute(widgetId)
      }
      catch (err) {
        settleReject(err as Error)
      }
    })
  }

  return { siteKey, isConfigured, getToken }
}
