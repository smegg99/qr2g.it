// app/utils/api.ts
// Turns generated client results into one shape the feature composables can test without HTTP.
import type { Error as ApiError } from '~/generated/api/types.gen'

export type { ApiError }

export interface ApiResult<T> {
  status: number
  data?: T
  error?: ApiError
}

/** A failure the request could not even report: offline, blocked or aborted. */
export const NETWORK_ERROR: ApiError = { code: 'network', message: 'errors.network' }

interface GeneratedResult<T> {
  data?: T
  error?: unknown
  response?: Response
}

/** Normalizes a generated call; transport failures become the network envelope. */
export async function call<T>(run: () => Promise<GeneratedResult<T>>): Promise<ApiResult<T>> {
  try {
    const { data, error, response } = await run()
    const status = response?.status ?? 0
    if (response?.ok && data !== undefined) return { status, data }
    const envelope = error as Partial<ApiError> | undefined
    if (envelope && typeof envelope.code === 'string' && typeof envelope.message === 'string') {
      return { status, error: { code: envelope.code, message: envelope.message, details: envelope.details } }
    }
    return { status, error: status ? { code: 'unknown', message: 'errors.unknown' } : NETWORK_ERROR }
  }
  catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err
    return { status: 0, error: NETWORK_ERROR }
  }
}
