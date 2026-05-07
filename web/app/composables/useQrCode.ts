// app/composables/useQrCode.ts
import QRCode from 'qrcode'

export type QrErrorLevel = 'L' | 'M' | 'Q' | 'H'

export interface QrOptions {
  text: string
  size: number
  margin: number
  foreground: string
  background: string
  errorCorrection: QrErrorLevel
  cornerRadius: number
}

export interface QrSnapshot {
  svg: string
  modules: number
  pixelSize: number
}

function clampPositive(n: number, min: number, max: number, fallback: number): number {
  if (!Number.isFinite(n)) return fallback
  return Math.max(min, Math.min(max, Math.round(n)))
}

interface BuildResult {
  svg: string
  modules: number
  total: number
  pixelSize: number
}

function pixelDimensions(requested: number, total: number) {
  const cellPx = Math.max(1, Math.round(requested / total))
  return { cellPx, size: cellPx * total }
}

function buildSvg(opts: QrOptions, requestedSize: number): BuildResult {
  const requested = clampPositive(requestedSize, 64, 4096, 512)
  const margin = clampPositive(opts.margin, 0, 16, 2)
  const cornerPct = Math.max(0, Math.min(50, opts.cornerRadius || 0))

  const matrix = QRCode.create(opts.text || ' ', { errorCorrectionLevel: opts.errorCorrection })
  const modules = matrix.modules.size
  const total = modules + margin * 2

  const { size } = pixelDimensions(requested, total)
  const fg = opts.foreground
  const bg = opts.background

  // Build a single <path> with horizontal-run merging. The viewBox is in
  // module units and rect coordinates are integers, so adjacent runs share an
  // exact integer edge -- no sub-pixel gaps. shape-rendering=crispEdges keeps
  // edges sharp, and the rasterization path snaps output to an integer
  // multiple of `total` so each module is exactly N pixels.
  const segments: string[] = []
  const data = matrix.modules.data
  for (let y = 0; y < modules; y++) {
    let runStart = -1
    for (let x = 0; x < modules; x++) {
      const v = data[y * modules + x]
      if (v) {
        if (runStart < 0) runStart = x
      }
      else if (runStart >= 0) {
        segments.push(runPath(runStart + margin, y + margin, x - runStart))
        runStart = -1
      }
    }
    if (runStart >= 0) {
      segments.push(runPath(runStart + margin, y + margin, modules - runStart))
    }
  }

  const dAttr = segments.join('')
  const radius = (cornerPct / 100) * total
  const clipDefs = radius > 0
    ? `<defs><clipPath id="qrclip"><rect width="${total}" height="${total}" rx="${radius}" ry="${radius}"/></clipPath></defs>`
    : ''
  const groupOpen = radius > 0 ? '<g clip-path="url(#qrclip)">' : '<g>'

  const head = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${total} ${total}" width="${size}" height="${size}" shape-rendering="crispEdges">`
  const body = `${clipDefs}${groupOpen}<rect width="${total}" height="${total}" fill="${bg}"/><path d="${dAttr}" fill="${fg}"/></g>`

  return {
    svg: `${head}${body}</svg>`,
    modules,
    total,
    pixelSize: size,
  }
}

function runPath(x: number, y: number, w: number): string {
  return `M${x} ${y}h${w}v1h-${w}z`
}

async function rasterize(svg: string, size: number, mime: string, quality?: number): Promise<Blob> {
  const blob = new Blob([svg], { type: 'image/svg+xml;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  try {
    const img = new Image()
    img.decoding = 'async'
    await new Promise<void>((resolve, reject) => {
      img.onload = () => resolve()
      img.onerror = reject
      img.src = url
    })
    const canvas = document.createElement('canvas')
    canvas.width = size
    canvas.height = size
    const ctx = canvas.getContext('2d')!
    ctx.imageSmoothingEnabled = false
    ctx.drawImage(img, 0, 0, size, size)
    return await canvasToBlob(canvas, mime, quality)
  }
  finally {
    URL.revokeObjectURL(url)
  }
}

function canvasToBlob(canvas: HTMLCanvasElement, mime: string, quality?: number): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (b) => {
        if (b) resolve(b)
        else reject(new Error('canvas toBlob returned null'))
      },
      mime,
      quality,
    )
  })
}

async function rasterToBmp(svg: string, size: number, background: string): Promise<Blob> {
  const blob = new Blob([svg], { type: 'image/svg+xml;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  try {
    const img = new Image()
    img.decoding = 'async'
    await new Promise<void>((resolve, reject) => {
      img.onload = () => resolve()
      img.onerror = reject
      img.src = url
    })
    const canvas = document.createElement('canvas')
    canvas.width = size
    canvas.height = size
    const ctx = canvas.getContext('2d')!
    ctx.imageSmoothingEnabled = false
    ctx.fillStyle = background
    ctx.fillRect(0, 0, size, size)
    ctx.drawImage(img, 0, 0, size, size)
    const data = ctx.getImageData(0, 0, size, size)
    return encodeBmp(data)
  }
  finally {
    URL.revokeObjectURL(url)
  }
}

function encodeBmp(image: ImageData): Blob {
  const { width, height, data } = image
  const rowSize = Math.floor((24 * width + 31) / 32) * 4
  const pixelArraySize = rowSize * height
  const fileSize = 14 + 40 + pixelArraySize
  const buf = new ArrayBuffer(fileSize)
  const view = new DataView(buf)
  const u8 = new Uint8Array(buf)

  view.setUint16(0, 0x4D42, true)
  view.setUint32(2, fileSize, true)
  view.setUint32(6, 0, true)
  view.setUint32(10, 14 + 40, true)
  view.setUint32(14, 40, true)
  view.setInt32(18, width, true)
  view.setInt32(22, height, true)
  view.setUint16(26, 1, true)
  view.setUint16(28, 24, true)
  view.setUint32(30, 0, true)
  view.setUint32(34, pixelArraySize, true)
  view.setInt32(38, 2835, true)
  view.setInt32(42, 2835, true)
  view.setUint32(46, 0, true)
  view.setUint32(50, 0, true)

  let offset = 14 + 40
  for (let y = height - 1; y >= 0; y--) {
    let rowOffset = offset
    for (let x = 0; x < width; x++) {
      const i = (y * width + x) * 4
      u8[rowOffset++] = data[i + 2] ?? 0
      u8[rowOffset++] = data[i + 1] ?? 0
      u8[rowOffset++] = data[i] ?? 0
    }
    offset += rowSize
  }
  return new Blob([buf], { type: 'image/bmp' })
}

export function useQrCode() {
  const snapshot = ref<QrSnapshot | null>(null)
  const generating = ref(false)
  const error = ref<string | null>(null)

  async function build(opts: QrOptions): Promise<QrSnapshot | null> {
    error.value = null
    generating.value = true
    try {
      const { svg, modules, pixelSize } = buildSvg(opts, opts.size)
      const result: QrSnapshot = { svg, modules, pixelSize }
      snapshot.value = result
      return result
    }
    catch (err) {
      error.value = String(err)
      return null
    }
    finally {
      generating.value = false
    }
  }

  async function exportSvg(opts: QrOptions): Promise<Blob> {
    const { svg } = buildSvg(opts, opts.size)
    return new Blob([svg], { type: 'image/svg+xml;charset=utf-8' })
  }

  async function exportPng(opts: QrOptions): Promise<Blob> {
    const { svg, pixelSize } = buildSvg(opts, opts.size)
    return rasterize(svg, pixelSize, 'image/png')
  }

  async function exportJpg(opts: QrOptions): Promise<Blob> {
    const { svg, pixelSize } = buildSvg(opts, opts.size)
    return rasterize(svg, pixelSize, 'image/jpeg', 0.95)
  }

  async function exportBmp(opts: QrOptions): Promise<Blob> {
    const { svg, pixelSize } = buildSvg(opts, opts.size)
    return rasterToBmp(svg, pixelSize, opts.background)
  }

  function downloadBlob(blob: Blob, filename: string) {
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    a.remove()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
  }

  return {
    snapshot,
    generating,
    error,
    build,
    exportSvg,
    exportPng,
    exportJpg,
    exportBmp,
    downloadBlob,
  }
}
