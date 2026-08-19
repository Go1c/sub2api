import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const viewPath = resolve(dirname(fileURLToPath(import.meta.url)), '../HomeView.vue')
const viewSource = readFileSync(viewPath, 'utf8')

function homeNavItemsBlock(): string {
  const match = viewSource.match(/const navItems = computed<NavItem\[\]>\(\(\) => \[([\s\S]*?)\]\)/)
  return match?.[1] || ''
}

describe('HomeView navigation', () => {
  it('replaces features navigation with the public model market page', () => {
    const navItems = homeNavItemsBlock()

    expect(navItems).toContain("key: 'models'")
    expect(navItems).toContain("target: '/models'")
    expect(navItems).not.toContain("target: '#features'")
    expect(navItems).not.toContain("key: 'features'")
  })

  it('links status navigation to the standalone public status page', () => {
    const navItems = homeNavItemsBlock()

    expect(navItems).toContain("key: 'status'")
    expect(navItems).toContain("target: '/status'")
    expect(navItems).not.toContain("target: '#status'")
  })

  it('does not keep the removed homepage pricing anchor in navigation', () => {
    const navItems = homeNavItemsBlock()

    expect(navItems).not.toContain("key: 'pricing'")
    expect(navItems).not.toContain("target: '#pricing'")
  })

  it('replaces support with the Image2 generator authenticated handoff link', () => {
    const navItems = homeNavItemsBlock()

    expect(navItems).toContain("key: 'image2'")
    expect(navItems).toContain('target: image2LoginHandoffTarget')
    expect(navItems).not.toContain("target: 'https://img.lumio.games/'")
    expect(navItems).not.toContain('external: true, dim: true')
    expect(navItems).not.toContain("key: 'support'")
    expect(navItems).not.toContain("target: '#support'")
  })

  it('adds a Codex download outbound link next to the model market', () => {
    const navItems = homeNavItemsBlock()

    expect(viewSource).toContain("from '@/constants/codexDownload'")
    expect(navItems).toContain("key: 'codexDownload'")
    expect(navItems).toContain('target: CODEX_DOWNLOAD_URL')
    expect(navItems).toContain('external: true')
    expect(navItems.indexOf("key: 'codexDownload'")).toBeGreaterThan(navItems.indexOf("key: 'models'"))
    expect(navItems.indexOf("key: 'codexDownload'")).toBeLessThan(navItems.indexOf("key: 'status'"))
  })
})
