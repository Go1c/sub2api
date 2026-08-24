import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { afterEach, describe, expect, it } from 'vitest'

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../..')
const spaIndexPath = resolve(frontendRoot, 'index.html')
const docsIndexPath = resolve(frontendRoot, 'public/docs/index.html')

const UMAMI_SRC = 'https://data.lumio.games/script.js'
const UMAMI_WEBSITE_ID = '423c8276-e57c-4e5b-81e6-63711a7fd1a5'
const OFFICIAL_SCRIPT_TAG =
  `<script defer src="${UMAMI_SRC}" data-website-id="${UMAMI_WEBSITE_ID}"></script>`

function readSpaIndex(): string {
  return readFileSync(spaIndexPath, 'utf8')
}

function readDocsIndex(): string {
  return readFileSync(docsIndexPath, 'utf8')
}

function extractNonceInlineScripts(html: string): string[] {
  const scripts: string[] = []
  const re = /<script\b([^>]*)>([\s\S]*?)<\/script>/gi
  let match: RegExpExecArray | null
  while ((match = re.exec(html)) !== null) {
    const attrs = match[1]
    const body = match[2].trim()
    if (!body) continue
    if (/nonce\s*=\s*["']__CSP_NONCE_VALUE__["']/i.test(attrs)) {
      scripts.push(body)
    }
  }
  return scripts
}

function countOfficialStaticTags(html: string): number {
  const re = /<script\b[^>]*>/gi
  const tags = html.match(re) ?? []
  return tags.filter((tag) => {
    const hasSrc = tag.includes(UMAMI_SRC)
    const hasId = tag.includes(UMAMI_WEBSITE_ID)
    const hasDefer = /\bdefer\b/i.test(tag)
    return hasSrc && hasId && hasDefer
  }).length
}

function installedUmamiScripts(): HTMLScriptElement[] {
  return Array.from(document.querySelectorAll('script')).filter((el) => {
    return el.getAttribute('src') === UMAMI_SRC
  })
}

function runSpaInline(pathname: string): void {
  const html = readSpaIndex()
  const inlines = extractNonceInlineScripts(html)
  expect(inlines.length).toBeGreaterThan(0)
  Object.defineProperty(window, 'location', {
    configurable: true,
    value: { pathname }
  })
  const runner = new Function(inlines.join('\n;\n'))
  runner()
}

describe('Umami public tracking HTML', () => {
  afterEach(() => {
    document.head.innerHTML = ''
    document.body.innerHTML = ''
  })

  it('SPA index does not embed the official script unconditionally', () => {
    expect(countOfficialStaticTags(readSpaIndex())).toBe(0)
  })

  it('SPA index uses a nonce inline that inserts the official deferred script off /admin', () => {
    const html = readSpaIndex()
    const inlines = extractNonceInlineScripts(html)
    expect(inlines.length).toBeGreaterThan(0)
    const joined = inlines.join('\n')
    expect(joined).toContain(UMAMI_SRC)
    expect(joined).toContain(UMAMI_WEBSITE_ID)
    expect(joined).toMatch(/defer/)
    expect(joined).toMatch(/\/admin/)
  })

  it('direct /admin and /admin/* do not insert script.js', () => {
    for (const path of ['/admin', '/admin/dashboard', '/admin/users']) {
      document.head.innerHTML = ''
      runSpaInline(path)
      expect(installedUmamiScripts(), path).toHaveLength(0)
    }
  })

  it('public first-load paths insert one official deferred Umami script', () => {
    for (const path of ['/', '/home', '/login', '/docs/', '/register']) {
      document.head.innerHTML = ''
      runSpaInline(path)
      const scripts = installedUmamiScripts()
      expect(scripts, path).toHaveLength(1)
      expect(scripts[0].defer).toBe(true)
      expect(scripts[0].getAttribute('src')).toBe(UMAMI_SRC)
      expect(scripts[0].getAttribute('data-website-id')).toBe(UMAMI_WEBSITE_ID)
    }
  })

  it('docs index uses the official script tag and no Umami inline', () => {
    const html = readDocsIndex()
    expect(html).toContain(OFFICIAL_SCRIPT_TAG)
    expect(countOfficialStaticTags(html)).toBe(1)
    const inlineScripts = [...html.matchAll(/<script\b([^>]*)>([\s\S]*?)<\/script>/gi)]
      .map((m) => m[2].trim())
      .filter(Boolean)
      .filter((body) => body.includes('data.lumio.games') || body.includes(UMAMI_WEBSITE_ID))
    expect(inlineScripts).toEqual([])
    expect(html).toContain('本服务面向全球，包括中国大陆')
  })
})
