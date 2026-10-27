// app/composables/useLatched.ts
import type { Ref } from 'vue'

/** The source's last non-empty value, so content can stay visible while its container closes. */
export function useLatched<T>(source: () => T | null | undefined): Ref<T | null> {
  const last = ref(source() ?? null) as Ref<T | null>
  watch(source, (next) => {
    if (next) last.value = next
  })
  return last
}
