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

function ensureContainer(): HTMLElement {
  const el = document.getElementById(CONTAINER_ID)
  if (!el) throw new Error('turnstile: container not mounted')
  return el
}

export function useTurnstile() {
  const siteKey = String(useRuntimeConfig().public.turnstileSiteKey || '')
  const isConfigured = siteKey !== ''
  let cancelCurrent: ((reason: Error) => void) | undefined

  onBeforeUnmount(() => cancelCurrent?.(new DOMException('turnstile: cancelled', 'AbortError')))

  function getToken(signal?: AbortSignal): Promise<string | null> {
    if (!isConfigured || typeof window === 'undefined') return Promise.resolve(null)
    cancelCurrent?.(new Error('turnstile: superseded by a new request'))

    return new Promise<string>((resolve, reject) => {
      let widget: string | undefined
      let api: TurnstileWindow['turnstile']
      let done = false
      let failures = 0

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

        widget = api.render(ensureContainer(), {
          'sitekey': siteKey,
          'execution': 'execute',
          'appearance': 'execute',
          'action': ACTION,
          'retry': 'auto',
          'retry-interval': 1000,
          'refresh-expired': 'auto',
          'callback': token => finish(token),
          'error-callback': (error) => {
            if (done) return false
            const code = String(error)
            const retryable = /^(300|600)/.test(code) || ['200500', '110600', '110620'].includes(code)
            const terminal = !retryable || ++failures >= 3
            if (terminal) finish(undefined, new Error(`turnstile: ${code}`))
            return !terminal
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
