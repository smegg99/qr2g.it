// app/composables/useShortLink.ts
import type { ApiErrorCode, ApiErrorResponse, LinkResponse } from '~/types/api'

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

const GITHUB_HOST = 'github.com'
const GIT_SUFFIX = '.git'
const MAX_OWNER_LEN = 39
const MAX_REPO_LEN = 100
const SHORT_RE = /^\s*([^/\s]+)\/([^/\s]+)\s*$/
const RESERVED_OWNER_SEGMENTS = new Set([
  'about',
  'account',
  'admin',
  'api',
  'apps',
  'assets',
  'blog',
  'business',
  'contact',
  'dashboard',
  'discussions',
  'enterprise',
  'events',
  'explore',
  'features',
  'gist',
  'home',
  'issues',
  'login',
  'logout',
  'marketplace',
  'new',
  'notifications',
  'organizations',
  'pricing',
  'pulls',
  'raw',
  'readme',
  'search',
  'security',
  'sessions',
  'settings',
  'signup',
  'site',
  'sponsors',
  'stars',
  'support',
  'trending',
  'tree',
  'users',
])

export interface ParsedRepo {
  owner: string
  name: string
  canonical: string
}

export function parseRepoInput(value: string): ParsedRepo | null {
  if (!value) return null
  const v = value.trim()
  if (v === '') return null

  if (v.includes('://')) return parseFullGithubUrl(v)
  if (v.toLowerCase().startsWith(`${GITHUB_HOST}/`)) return parseRepoPath(v.slice(GITHUB_HOST.length))

  const m = v.match(SHORT_RE)
  if (!m) return null
  return parseOwnerRepo(m[1] ?? '', m[2] ?? '')
}

export function isLikelyGithubRepoUrl(value: string): boolean {
  return parseRepoInput(value) !== null
}

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
    loading.value = true
    abort?.abort()
    abort = new AbortController()

    const parsed = parseRepoInput(input.url)
    if (!parsed) {
      error.value = { code: 'invalid_url' }
      loading.value = false
      abort = null
      return null
    }

    let turnstileToken = input.turnstileToken ?? ''
    if (!turnstileToken && turnstile.isConfigured) {
      try {
        const t = await turnstile.getToken()
        if (!t) throw new Error('turnstile: empty token')
        turnstileToken = t
      }
      catch (err) {
        error.value = { code: 'turnstile_unavailable', message: String(err) }
        loading.value = false
        abort = null
        return null
      }
    }

    try {
      const res = await fetch(`${config.public.apiBase}/links`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          url: parsed.canonical,
          turnstileToken,
        }),
        signal: abort.signal,
      })

      const text = await res.text()
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
      if ((err as Error)?.name === 'AbortError') return null
      error.value = { code: 'network', message: String(err) }
      return null
    }
    finally {
      loading.value = false
      abort = null
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

  onBeforeUnmount(() => abort?.abort())

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

function parseFullGithubUrl(value: string): ParsedRepo | null {
  let url: URL
  try {
    url = new URL(value)
  }
  catch {
    return null
  }
  if (url.protocol.toLowerCase() !== 'https:') return null
  if (url.username || url.password || url.port) return null
  if (url.hostname.toLowerCase() !== GITHUB_HOST) return null
  if (url.search || url.hash) return null
  return parseRepoPath(url.pathname)
}

function parseRepoPath(path: string): ParsedRepo | null {
  const trimmedPath = path.replace(/^\//, '').replace(/\/$/, '')
  const parts = trimmedPath.split('/')
  if (parts.length !== 2) return null
  return parseOwnerRepo(parts[0] ?? '', parts[1] ?? '')
}

function parseOwnerRepo(owner: string, repo: string): ParsedRepo | null {
  const name = repo.toLowerCase().endsWith(GIT_SUFFIX)
    ? repo.slice(0, -GIT_SUFFIX.length)
    : repo
  if (!validOwner(owner) || RESERVED_OWNER_SEGMENTS.has(owner.toLowerCase())) return null
  if (!validRepo(name)) return null
  return {
    owner,
    name,
    canonical: `https://github.com/${owner}/${name}`,
  }
}

function validOwner(value: string): boolean {
  if (value.length === 0 || value.length > MAX_OWNER_LEN) return false
  if (value.startsWith('-') || value.endsWith('-') || value.includes('--')) return false
  return /^[A-Za-z0-9-]+$/.test(value)
}

function validRepo(value: string): boolean {
  if (value.length === 0 || value.length > MAX_REPO_LEN) return false
  if (value === '.' || value === '..') return false
  return /^[A-Za-z0-9._-]+$/.test(value)
}
