export type SiteMessageContentPart =
  | { type: 'text'; text: string }
  | { type: 'image'; url: string }

export function publicHttpsImageUrl(value: string): string {
  const trimmed = value.trim()
  if (!trimmed) {
    return ''
  }

  try {
    const parsed = new URL(trimmed)
    if (parsed.protocol !== 'https:' || !parsed.host || parsed.username || parsed.password) {
      return ''
    }
    return parsed.toString()
  } catch {
    return ''
  }
}

export function safeLotteryPromoImageUrl(value: string): string {
  const trimmed = value.trim()
  if (trimmed.startsWith('/') && !trimmed.startsWith('//') && !trimmed.includes('\\')) {
    return trimmed
  }
  return publicHttpsImageUrl(trimmed)
}

export function splitSiteMessageContent(content: string): SiteMessageContentPart[] {
  const lines = content.split('\n')
  const parts: SiteMessageContentPart[] = []
  let textLines: string[] = []

  const flushText = () => {
    if (textLines.length === 0) {
      return
    }
    parts.push({ type: 'text', text: textLines.join('\n') })
    textLines = []
  }

  for (const line of lines) {
    const trimmed = line.trim()
    const imageUrl = publicHttpsImageUrl(trimmed)
    if (imageUrl && !trimmed.includes(' ')) {
      flushText()
      parts.push({ type: 'image', url: imageUrl })
      continue
    }
    textLines.push(line)
  }

  flushText()
  return parts
}
