// app/composables/useAsset.ts
// Preview and export both ask the server renderer, so what is shown is what is saved.
import type { Ref } from 'vue'
import { renderLinkAsset } from '~/generated/api/sdk.gen'
import type { RenderLinkAssetData } from '~/generated/api/types.gen'
import { NETWORK_ERROR, type ApiError } from '~/utils/api'
import { assetQuery, type Design, type Format } from '~/utils/design'

export interface AssetInfo {
  width: number
  height: number
  type: string
  bytes: number
}

interface Rendered {
  blob: Blob
  info: AssetInfo
}

async function render(slug: string, design: Design, format: Format, download: boolean, signal: AbortSignal): Promise<Rendered> {
  const query = { ...assetQuery(design, format), ...(download ? { download: 'true' } : {}) }
  const { data, error, response } = await renderLinkAsset({
    path: { slug },
    query: query as unknown as RenderLinkAssetData['query'],
    parseAs: 'blob',
    signal,
  })
  if (!response?.ok || !(data instanceof Blob)) {
    const envelope = error as Partial<ApiError> | undefined
    throw envelope?.code ? envelope : (response ? { code: 'unknown', message: 'errors.unknown' } : NETWORK_ERROR)
  }
  return {
    blob: data,
    info: {
      width: Number(response.headers.get('X-Asset-Width')) || 0,
      height: Number(response.headers.get('X-Asset-Height')) || 0,
      type: data.type || response.headers.get('Content-Type') || '',
      bytes: data.size,
    },
  }
}

function asError(err: unknown): ApiError {
  const e = err as Partial<ApiError> | undefined
  return e?.code && e.message ? { code: e.code, message: e.message, details: e.details } : NETWORK_ERROR
}

/** A debounced SVG preview of the current design; older renders never replace newer ones. */
export function useAssetPreview(slug: Ref<string | null>, design: Ref<Design>) {
  const src = ref<string | null>(null)
  const info = ref<AssetInfo | null>(null)
  const loading = ref(false)
  const error = ref<ApiError | null>(null)
  let owner: AbortController | null = null
  let timer: ReturnType<typeof setTimeout> | undefined

  function show(url: string | null) {
    if (src.value) URL.revokeObjectURL(src.value)
    src.value = url
  }

  async function refresh() {
    owner?.abort()
    const current = slug.value
    if (!current) {
      show(null)
      info.value = null
      return
    }
    const request = new AbortController()
    owner = request
    loading.value = true
    try {
      const rendered = await render(current, design.value, 'svg', false, request.signal)
      if (owner !== request) return
      show(URL.createObjectURL(rendered.blob))
      info.value = rendered.info
      error.value = null
    }
    catch (err) {
      if (owner === request && !request.signal.aborted) error.value = asError(err)
    }
    finally {
      if (owner === request) loading.value = false
    }
  }

  watch([slug, () => JSON.stringify(assetQuery(design.value, 'svg'))], () => {
    clearTimeout(timer)
    timer = setTimeout(refresh, 180)
  }, { immediate: true })

  onBeforeUnmount(() => {
    clearTimeout(timer)
    owner?.abort()
    show(null)
  })

  return { src, info, loading, error, refresh }
}

/** Renders the chosen format and saves it under the server's filename. */
export function useAssetExport() {
  const exporting = ref(false)
  const error = ref<ApiError | null>(null)
  const last = ref<AssetInfo | null>(null)
  let owner: AbortController | null = null

  async function download(slug: string, design: Design, filename: string): Promise<boolean> {
    owner?.abort()
    const request = new AbortController()
    owner = request
    exporting.value = true
    error.value = null
    try {
      const rendered = await render(slug, design, design.format, true, request.signal)
      if (owner !== request) return false
      last.value = rendered.info
      const url = URL.createObjectURL(rendered.blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      a.remove()
      setTimeout(() => URL.revokeObjectURL(url), 1000)
      return true
    }
    catch (err) {
      if (owner === request && !request.signal.aborted) error.value = asError(err)
      return false
    }
    finally {
      if (owner === request) exporting.value = false
    }
  }

  onBeforeUnmount(() => owner?.abort())

  return { exporting, error, last, download }
}
