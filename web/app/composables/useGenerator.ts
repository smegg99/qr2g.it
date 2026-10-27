// app/composables/useGenerator.ts
// The generator's server state: one owned request at a time, so reset, a new
// repository or navigation can never let a stale verification or response win.
import type { Link } from '~/generated/api/types.gen'
import type { ApiError, ApiResult } from '../utils/api.ts'
import { parseRepoInput } from '../utils/repository.ts'

export interface GeneratorApi {
  createLink(body: { url: string, turnstileToken?: string }, signal: AbortSignal): Promise<ApiResult<Link>>
  getLink(slug: string, signal: AbortSignal): Promise<ApiResult<Link>>
}

/** What this request did; it is not a permanent property of the repository. */
export type Outcome = 'created' | 'existing'

/** The step the user is waiting on; only stages the app can observe. */
export type Stage = 'idle' | 'verifying' | 'requesting'

const POLL_ATTEMPTS = 4
const POLL_MS = 2500

const local = (code: string): ApiError => ({ code, message: `errors.${code}` })

export function useGenerator(api: GeneratorApi) {
  const turnstile = useTurnstile()
  const link = ref<Link | null>(null)
  const outcome = ref<Outcome | null>(null)
  const stage = ref<Stage>('idle')
  const error = ref<ApiError | null>(null)
  let owner: AbortController | null = null
  // Advances with every claim, so follow-up reads belong to one result only.
  let generation = 0
  let poll: ReturnType<typeof setTimeout> | undefined

  const loading = computed(() => stage.value !== 'idle')

  /** Takes ownership; every older request and its callbacks become inert. */
  function claim(): AbortController {
    owner?.abort()
    clearTimeout(poll)
    generation++
    const request = new AbortController()
    owner = request
    return request
  }

  function settle(request: AbortController) {
    if (owner === request) {
      stage.value = 'idle'
      owner = null
    }
  }

  function accept(request: AbortController, result: ApiResult<Link>, kind: Outcome | null): Link | null {
    if (request.signal.aborted) return null
    if (!result.data) {
      error.value = result.error ?? local('unknown')
      return null
    }
    link.value = result.data
    outcome.value = kind
    error.value = null
    watchPolicy(result.data.slug)
    return result.data
  }

  async function create(input: string): Promise<Link | null> {
    const parsed = parseRepoInput(input)
    if (!parsed) {
      claim()
      stage.value = 'idle'
      error.value = local('invalid_url')
      return null
    }
    const request = claim()
    error.value = null
    try {
      let token = ''
      if (turnstile.isConfigured) {
        stage.value = 'verifying'
        try {
          token = (await turnstile.getToken(request.signal)) ?? ''
        }
        catch {
          token = ''
        }
        if (request.signal.aborted) return null
        if (!token) {
          error.value = local('turnstile_unavailable')
          return null
        }
      }
      stage.value = 'requesting'
      const result = await api.createLink({ url: parsed.canonical, turnstileToken: token || undefined }, request.signal)
      return accept(request, result, result.status === 201 ? 'created' : 'existing')
    }
    catch {
      if (!request.signal.aborted) error.value = local('network')
      return null
    }
    finally {
      settle(request)
    }
  }

  /** Follows a pending first policy check for a few observable rounds. */
  function watchPolicy(slug: string, attempt = 0) {
    clearTimeout(poll)
    if (attempt >= POLL_ATTEMPTS || link.value?.policy.listing !== 'pending') return
    const watcher = generation
    poll = setTimeout(async () => {
      if (generation !== watcher || link.value?.slug !== slug) return
      const request = new AbortController()
      const result = await api.getLink(slug, request.signal)
      if (generation !== watcher || link.value?.slug !== slug || !result.data) return
      link.value = result.data
      watchPolicy(slug, attempt + 1)
    }, POLL_MS)
  }

  /** Drops repository-specific results; design choices live elsewhere and survive. */
  /** Reads the open link again, for its current state and facts; a failure keeps what is shown. */
  async function refresh() {
    const current = link.value
    if (!current) return
    const result = await api.getLink(current.slug, new AbortController().signal)
    if (result.data && link.value?.slug === current.slug) link.value = result.data
  }

  function reset() {
    claim().abort()
    owner = null
    link.value = null
    outcome.value = null
    error.value = null
    stage.value = 'idle'
  }

  function clearError() {
    error.value = null
  }

  onBeforeUnmount(reset)

  return { link, outcome, stage, loading, error, create, refresh, reset, clearError }
}
