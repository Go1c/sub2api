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
  })

  it('ships a plain-text llms-full.txt that names LumioAPI', () => {
    expect(llmsFull.startsWith('# LumioAPI')).toBe(true)
    expect(llmsFull).toContain('https://api.lumio.games/')
    expect(llmsFull).not.toMatch(/<!doctype html>/i)
  })
})
