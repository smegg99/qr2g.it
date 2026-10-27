// app/composables/useShortLink.ts
import type { ApiErrorCode, ApiErrorResponse, LinkResponse } from '~/types/api'
import { parseRepoInput } from '../utils/repository.ts'

export interface CreateLinkInput {
  url: string
  turnstileToken?: string
}

export interface ShortLinkErrorState {
  code: ApiErrorCode
  status?: number
  message?: string
}

const KNOWN_ERROR_CODES = new Set<ApiErrorCode>([
  'invalid_body',
  'invalid_url',
  'rate_limited',
  'turnstile_required',
  'turnstile_failed',
  'turnstile_unavailable',
  'repo_not_found',
  'repo_private',
  'github_auth',
  'github_rate_limited',
  'github_unavailable',
  'verification_failed',
  'internal',
])

export function useShortLink() {
  const config = useRuntimeConfig()
  const turnstile = useTurnstile()
  const link = ref<LinkResponse | null>(null)
  const loading = ref(false)
  const error = ref<ShortLinkErrorState | null>(null)
  let abort: AbortController | null = null

  async function create(input: CreateLinkInput): Promise<LinkResponse | null> {
    if (loading.value) return null
    error.value = null
    const parsed = parseRepoInput(input.url)
    if (!parsed) {
      error.value = { code: 'invalid_url' }
      return null
    }

    const request = new AbortController()
    abort = request
    loading.value = true
    try {
      let turnstileToken = input.turnstileToken ?? ''
      if (!turnstileToken && turnstile.isConfigured) {
        try {
          const token = await turnstile.getToken(request.signal)
          if (!token) throw new Error('turnstile: empty token')
          turnstileToken = token
        }
        catch (err) {
          if (!request.signal.aborted) error.value = { code: 'turnstile_unavailable', message: String(err) }
          return null
        }
      }
      if (request.signal.aborted) return null

      const res = await fetch(`${config.public.apiBase}/links`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url: parsed.canonical, turnstileToken }),
        signal: request.signal,
      })
      const text = await res.text()
      if (request.signal.aborted) return null
      const body = text ? safeJson(text) : null

      if (!res.ok) {
        const code = (body as ApiErrorResponse | null)?.error
        const message = (body as ApiErrorResponse | null)?.message
        error.value = {
          code: KNOWN_ERROR_CODES.has(code as ApiErrorCode) ? (code as ApiErrorCode) : 'unknown',
          status: res.status,
          message,
        }
        return null
      }
      link.value = body as LinkResponse
      return link.value
    }
    catch (err) {
      if (!request.signal.aborted) error.value = { code: 'network', message: String(err) }
      return null
    }
    finally {
      if (abort === request) {
        loading.value = false
        abort = null
      }
    }
  }

  function reset() {
    abort?.abort()
    abort = null
    link.value = null
    error.value = null
    loading.value = false
  }

  function clearError() {
    error.value = null
  }

  onBeforeUnmount(reset)

  return { link, loading, error, create, reset, clearError }
}

function safeJson(text: string): unknown {
  try {
    return JSON.parse(text)
  }
  catch {
    return null
  }
}
