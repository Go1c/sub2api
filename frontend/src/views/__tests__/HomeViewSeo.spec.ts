import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const indexHtml = readFileSync(resolve(here, '../../../index.html'), 'utf8')
const llmsFull = readFileSync(resolve(here, '../../../public/llms-full.txt'), 'utf8')

describe('public homepage SEO', () => {
  it('ships a Chinese title, crawler-readable Chinese H1, and search engine verification', () => {
    expect(indexHtml).toContain('<title>LumioAPI · AI API 中转与管理平台</title>')
    expect(indexHtml).toContain('<h1>LumioAPI · AI API 中转与管理平台</h1>')
    expect(indexHtml).not.toContain('seo-crawler-intro')
    expect(indexHtml).not.toMatch(/display\s*:\s*none/i)
    const appOpen = indexHtml.indexOf('<div id="app">')
    const appClose = indexHtml.indexOf('</div>', appOpen)
    expect(appOpen).toBeGreaterThan(-1)
    expect(appClose).toBeGreaterThan(appOpen)
    const appInner = indexHtml.slice(appOpen, appClose)
    expect(appInner).toContain('<h1>LumioAPI · AI API 中转与管理平台</h1>')
    expect(appInner).toContain(
      'LumioAPI 是 AI API 中转与管理平台，统一接入 Anthropic Claude、OpenAI GPT、Google Gemini 等主流模型。注册地址 https://api.lumio.games/register ，文档地址 https://api.lumio.games/docs/ 。'
    )
    expect(appInner).not.toMatch(/\bhidden\b/)
    expect(appInner).not.toMatch(/aria-hidden|visibility\s*:\s*hidden/i)
    expect(indexHtml).toContain(
      '<meta name="msvalidate.01" content="48232FF4A9EAB80D49C7A5AE2D009539" />'
    )
    expect(indexHtml).toContain(
      '<meta name="baidu-site-verification" content="codeva-zP7LfM4N1h" />'
    )
    expect(indexHtml).toContain(
      '<meta name="google-site-verification" content="YlFC2R5DY626I5yH2cA24zxqOOmciWqMHQcuJAw2El8" />'
    )
    expect(indexHtml).toContain(
      '<meta name="sogou_site_verification" content="5QR2ynT5Cb" />'
    )
    expect(indexHtml).toContain('<meta property="og:locale" content="zh_CN" />')
    expect(indexHtml).toContain(
      '<link rel="alternate" hreflang="zh-CN" href="https://api.lumio.games/" />'
    )
    expect(indexHtml).toContain(
      '<link rel="alternate" hreflang="x-default" href="https://api.lumio.games/" />'
    )
    expect(indexHtml).toContain('<meta property="og:image" content="https://api.lumio.games/logo.png" />')
  })

  it('ships a plain-text llms-full.txt that names LumioAPI', () => {
    expect(llmsFull.startsWith('# LumioAPI')).toBe(true)
    expect(llmsFull).toContain('https://api.lumio.games/')
    expect(llmsFull).not.toMatch(/<!doctype html>/i)
  })

  it('points llms.txt at the full-text file', () => {
    const llms = readFileSync(resolve(here, '../../../public/llms.txt'), 'utf8')
    expect(llms).toContain('https://api.lumio.games/llms-full.txt')
  })
})
