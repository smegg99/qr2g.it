// app/composables/useGeneratorApi.ts
import { createLink, getLink } from '~/generated/api/sdk.gen'
import { call } from '~/utils/api'
import type { GeneratorApi } from './useGenerator'

/** The generated client behind the generator's request interface. */
export function useGeneratorApi(): GeneratorApi {
  return {
    createLink: (body, signal) => call(() => createLink({ body, signal })),
    getLink: (slug, signal) => call(() => getLink({ path: { slug }, signal })),
  }
}
