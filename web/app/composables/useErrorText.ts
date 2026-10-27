// app/composables/useErrorText.ts
import type { ApiError } from '~/utils/api'

/** The translated message for an API error, or the generic one for a key this build does not know. */
export function useErrorText() {
  const { t, te } = useI18n()
  return (error: ApiError) => (te(error.message) ? t(error.message) : t('errors.unknown'))
}
