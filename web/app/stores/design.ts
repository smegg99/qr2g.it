// app/stores/design.ts
// Export appearance on this device. Separate from site appearance and from shared repository policy.
import { defineStore } from 'pinia'
import { DEFAULT_DESIGN, designFromObject, type Design } from '~/utils/design'
import { readStorage, writeStorage } from '~/utils/storage'

const DESIGN_KEY = 'qr2g.design'

function load(): Design {
  try {
    return designFromObject(JSON.parse(readStorage(DESIGN_KEY) ?? 'null'))
  }
  catch {
    return { ...DEFAULT_DESIGN }
  }
}

export const useDesignStore = defineStore('design', () => {
  const design = ref<Design>(import.meta.client ? load() : { ...DEFAULT_DESIGN })

  watch(design, value => writeStorage(DESIGN_KEY, JSON.stringify(value)), { deep: true })

  function apply(next: Design) {
    design.value = { ...next }
  }

  return { design, apply }
})
