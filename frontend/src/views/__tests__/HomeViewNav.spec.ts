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
  it('links status navigation to the standalone public status page', () => {
    const navItems = homeNavItemsBlock()

    expect(navItems).toContain("key: 'status'")
    expect(navItems).toContain("target: '/status'")
    expect(navItems).not.toContain("target: '#status'")
  })

  it('replaces support with the Image2 generator external link', () => {
    const navItems = homeNavItemsBlock()

    expect(navItems).toContain("key: 'image2'")
    expect(navItems).toContain("target: 'https://img.lumio.games/'")
    expect(navItems).toContain('external: true')
    expect(navItems).not.toContain("key: 'support'")
    expect(navItems).not.toContain("target: '#support'")
  })
})
