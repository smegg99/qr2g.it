// app/utils/repository.ts
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
