// app/composables/useTurnstile.ts

interface TurnstileWindow {
  turnstile?: {
    render: (el: HTMLElement | string, opts: TurnstileRenderOptions) => string
    reset: (id: string) => void
    remove: (id: string) => void
    execute: (id: string) => void
    getResponse: (id?: string) => string | undefined
  }
}

interface TurnstileRenderOptions {
  'sitekey': string
  'size'?: 'normal' | 'compact' | 'flexible'
  'execution'?: 'render' | 'execute'
  'appearance'?: 'always' | 'execute' | 'interaction-only'
  'action'?: string
  'retry'?: 'auto' | 'never'
  'refresh-expired'?: 'auto' | 'manual' | 'never'
  'callback'?: (token: string) => void
  'error-callback'?: (err: unknown) => void
  'expired-callback'?: () => void
  'timeout-callback'?: () => void
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

  scriptPromise = new Promise<void>((resolve, reject) => {
    const existing = document.querySelector<HTMLScriptElement>(`script[src^="${SCRIPT_URL.split('?')[0]}"]`)
    if (existing) {
      existing.addEventListener('load', () => resolve(), { once: true })
      existing.addEventListener('error', () => reject(new Error('turnstile: script load failed')), { once: true })
      return
    }
    const s = document.createElement('script')
    s.src = SCRIPT_URL
    s.async = true
    s.defer = true
    s.onload = () => resolve()
    s.onerror = () => reject(new Error('turnstile: script load failed'))
    document.head.appendChild(s)
  })
  return scriptPromise
}

function ensureContainer(): HTMLElement {
  let el = document.getElementById(CONTAINER_ID)
  if (el) return el
  el = document.createElement('div')
  el.id = CONTAINER_ID
  el.setAttribute('aria-hidden', 'true')
  Object.assign(el.style, {
    position: 'fixed',
    right: '0',
    bottom: '0',
    width: '0',
    height: '0',
    overflow: 'hidden',
    opacity: '0',
    pointerEvents: 'none',
    zIndex: '-1',
  } as Partial<CSSStyleDeclaration>)
  document.body.appendChild(el)
  return el
}

function settleResolve(token: string) {
  pendingResolve?.(token)
  pendingResolve = null
  pendingReject = null
}

function settleReject(err: Error) {
  pendingReject?.(err)
  pendingResolve = null
  pendingReject = null
}

export function useTurnstile() {
  const config = useRuntimeConfig()
  const siteKey = (config.public.turnstileSiteKey as string) || ''
  const isConfigured = !!siteKey

  async function getToken(): Promise<string | null> {
    if (!isConfigured) return null
    if (typeof window === 'undefined') return null

    try {
      await loadScript()
    }
    catch {
      return null
    }
    const turnstile = (window as unknown as TurnstileWindow).turnstile
    if (!turnstile) return null

    return new Promise<string>((resolve, reject) => {
      pendingReject?.(new Error('turnstile: superseded by a new request'))
      pendingResolve = resolve
      pendingReject = reject

      const onCallback = (token: string) => settleResolve(token)
      const onError = (err: unknown) => settleReject(new Error(`turnstile: ${String(err)}`))

      if (widgetId) {
        try {
          turnstile.reset(widgetId)
          turnstile.execute(widgetId)
        }
        catch (err) {
          settleReject(err as Error)
        }
        return
      }

      try {
        widgetId = turnstile.render(ensureContainer(), {
          'sitekey': siteKey,
          'execution': 'execute',
          'appearance': 'execute',
          'action': ACTION,
          'retry': 'auto',
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
