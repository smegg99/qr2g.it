// app/composables/useTurnstile.ts

interface TurnstileWindow extends Window {
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
  'theme'?: 'light' | 'dark' | 'auto'
  'language'?: string
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
export const TURNSTILE_HOST = 'qr2g-turnstile-host'

/** Which form a token is for; the server accepts a token only for its own action. */
export interface TurnstileTarget {
  action: 'create_link' | 'contact'
  container: string
}

const RETRY_MS = 1000

const CREATE_LINK: TurnstileTarget = { action: 'create_link', container: TURNSTILE_HOST }

let scriptPromise: Promise<void> | null = null

function loadScript(): Promise<void> {
  if (typeof window === 'undefined') return Promise.reject(new Error('turnstile: window not available'))
  const tw = window as TurnstileWindow
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

// The widget follows the site's theme and language, not the browser's.
function pageTheme(): 'light' | 'dark' {
  return document.querySelector('.v-application')?.classList?.contains('v-theme--dark') ? 'dark' : 'light'
}

function ensureContainer(id: string): HTMLElement {
  const el = document.getElementById(id)
  if (!el) throw new Error('turnstile: container not mounted')
  return el
}

export function useTurnstile() {
  const siteKey = String(useRuntimeConfig().public.turnstileSiteKey || '')
  const isConfigured = siteKey !== ''
  let cancelCurrent: ((reason: Error) => void) | undefined

  onBeforeUnmount(() => cancelCurrent?.(new DOMException('turnstile: cancelled', 'AbortError')))

  function getToken(signal?: AbortSignal, target: TurnstileTarget = CREATE_LINK): Promise<string | null> {
    if (!isConfigured || typeof window === 'undefined') return Promise.resolve(null)
    cancelCurrent?.(new Error('turnstile: superseded by a new request'))

    return new Promise<string>((resolve, reject) => {
      let widget: string | undefined
      let api: TurnstileWindow['turnstile']
      let done = false
      let failures = 0
      // Retries are ours, so no Cloudflare timer can reset a widget that was already removed.
      let retry: ReturnType<typeof setTimeout> | undefined

      function removeWidget() {
        const id = widget
        widget = undefined
        if (!id) return
        try {
          api?.remove(id)
        }
        catch {
          // The widget may already have been removed.
        }
      }

      function finish(token?: string, error?: unknown) {
        if (done) return
        done = true
        clearTimeout(retry)
        signal?.removeEventListener('abort', onAbort)
        if (cancelCurrent === cancel) cancelCurrent = undefined
        removeWidget()
        if (token !== undefined) resolve(token)
        else reject(error)
      }

      const cancel = (reason: Error) => finish(undefined, reason)
      const onAbort = () => cancel(new DOMException('turnstile: cancelled', 'AbortError'))
      cancelCurrent = cancel
      if (signal?.aborted) {
        onAbort()
        return
      }
      signal?.addEventListener('abort', onAbort, { once: true })

      async function mount() {
        for (let attempt = 0; attempt < 2; attempt++) {
          try {
            await loadScript()
            break
          }
          catch (error) {
            if (done) return
            if (attempt === 1) throw error
          }
        }
        if (done) return
        api = (window as TurnstileWindow).turnstile
        if (!api) throw new Error('turnstile: API unavailable')

        widget = api.render(ensureContainer(target.container), {
          'sitekey': siteKey,
          'execution': 'execute',
          // Drawn only when a person has to interact, at the width of its host.
          'appearance': 'interaction-only',
          'size': 'flexible',
          'theme': pageTheme(),
          'language': document.documentElement?.lang || 'auto',
          'action': target.action,
          'retry': 'never',
          'refresh-expired': 'auto',
          'callback': token => finish(token),
          'error-callback': (error) => {
            if (done) return true
            const code = String(error)
            const retryable = /^(300|600)/.test(code) || ['200500', '110600', '110620'].includes(code)
            if (!retryable || ++failures >= 3) {
              finish(undefined, new Error(`turnstile: ${code}`))
              return true
            }
            clearTimeout(retry)
            retry = setTimeout(() => {
              if (done || !widget) return
              try {
                // A reset widget waits for execute again, as on the first attempt.
                api?.reset(widget)
                api?.execute(widget)
              }
              catch {
                // The widget may already have been removed.
              }
            }, RETRY_MS)
            return true
          },
          'expired-callback': () => {
            if (!widget || done) return
            try {
              api?.reset(widget)
            }
            catch {
              // The widget may already have been removed.
            }
          },
          'timeout-callback': () => finish(undefined, new Error('turnstile: challenge timed out')),
          'unsupported-callback': () => finish(undefined, new Error('turnstile: browser unsupported')),
        })
        if (done) removeWidget()
        else api.execute(widget)
      }

      void mount().catch(error => finish(undefined, error))
    })
  }

  return { siteKey, isConfigured, getToken }
}
