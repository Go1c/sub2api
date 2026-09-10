import { sanitizeSvg } from '@/utils/sanitize'

export const ACCOUNT_IQ_TEST_MODE = 'iq' as const

const SVG_OPEN = /<svg\b/i
const FENCE_RE = /```(?:svg|xml|html)[^\n]*\n([\s\S]*?)```/gi
const TAG_RE = /<svg\b[^>]*\/>|<svg\b[^>]*>|<\/svg\s*>/gi

export function buildIqTestRequest(modelId: string, prompt: string) {
  return {
    model_id: modelId,
    prompt,
    mode: ACCOUNT_IQ_TEST_MODE
  }
}

/** 智商检测前端唯一入口：从模型回复抽出完整 SVG，做成 <img> 可用的 data URL。 */
export function applyIqTestOutput(text: string): string | null {
  const svg = extractCompleteSvg(text)
  if (!svg) return null
  const url = svgToImageUrl(svg)
  return url || null
}

export function extractCompleteSvg(text: string): string | null {
  if (!text) return null
  const normalized = stripBom(text)
  const fromFence = extractFromFences(normalized)
  if (fromFence) return fromFence
  return extractLastCompleteSvg(maybeUnescape(normalized))
}

export function iqSvgToImageUrl(svg: string): string | null {
  if (!svg) return null
  const url = svgToImageUrl(svg)
  return url || null
}

function svgToImageUrl(svg: string): string {
  const clean = sanitizeSvg(svg).trim()
  if (!clean) return ''
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(clean)}`
}

function stripBom(text: string): string {
  return text.replace(/^\uFEFF/, '')
}

function maybeUnescape(text: string): string {
  if (SVG_OPEN.test(text)) return text
  if (/&lt;svg\b/i.test(text)) {
    return text
      .replace(/&lt;/g, '<')
      .replace(/&gt;/g, '>')
      .replace(/&quot;/g, '"')
      .replace(/&apos;/g, "'")
      .replace(/&amp;/g, '&')
  }
  return text
}

function extractFromFences(text: string): string | null {
  let last: string | null = null
  const fenceRe = new RegExp(FENCE_RE.source, 'gi')
  let match: RegExpExecArray | null
  while ((match = fenceRe.exec(text)) !== null) {
    const svg = extractLastCompleteSvg(maybeUnescape(match[1] ?? ''))
    if (svg) last = svg
  }
  return last
}

function extractLastCompleteSvg(text: string): string | null {
  const haystack = maybeUnescape(text)
  const tagRe = new RegExp(TAG_RE.source, 'gi')
  const stack: number[] = []
  let last: string | null = null
  let match: RegExpExecArray | null
  while ((match = tagRe.exec(haystack))) {
    const token = match[0]
    if (/^<svg\b[^>]*\/>$/i.test(token)) {
      if (stack.length === 0) {
        last = haystack.slice(match.index, match.index + token.length)
      }
      continue
    }
    if (/^<svg/i.test(token)) {
      stack.push(match.index)
      continue
    }
    if (stack.length === 0) continue
    const start = stack.pop()!
    if (stack.length === 0) {
      last = haystack.slice(start, match.index + token.length)
    }
  }
  return last
}
