// app/types/api.ts
export interface LinkResponse {
  url: string
  slug: string
  id: number
  initialFullName: string
  currentFullName: string
  currentUrl: string
  state: 'active' | 'tombstoned' | 'stale' | string
  stateReason?: string
  verifiedAt?: string
  verifyAfter?: string
  createdAt: string
  githubNodeId: string
  githubDatabaseId: number
}

export interface ApiErrorResponse {
  error: string
  message: string
}

export type ApiErrorCode
  = | 'invalid_body'
    | 'invalid_url'
    | 'rate_limited'
    | 'turnstile_required'
    | 'turnstile_failed'
    | 'turnstile_unavailable'
    | 'repo_not_found'
    | 'repo_private'
    | 'github_auth'
    | 'github_rate_limited'
    | 'github_unavailable'
    | 'verification_failed'
    | 'internal'
    | 'network'
    | 'unknown'
