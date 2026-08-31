import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const homeViewSource = readFileSync(resolve(here, '../HomeView.vue'), 'utf8')
const legalNoticeSource = readFileSync(resolve(here, '../public/LegalNoticeView.vue'), 'utf8')

describe('HomeView compliance — removed homepage pricing', () => {
  it('removes the legacy homepage pricing table and public pricing API usage', () => {
    expect(homeViewSource).not.toContain('id="pricing"')
    expect(homeViewSource).not.toContain('copy.pricing')
    expect(homeViewSource).not.toContain('getPublicPricing')
    expect(homeViewSource).not.toContain('/pricing/public')
    expect(homeViewSource).not.toContain('formatCny')
  })

  it('keeps model market as the public model discovery entry', () => {
    expect(homeViewSource).toContain("key: 'models'")
    expect(homeViewSource).toContain("target: '/model-market'")
    expect(homeViewSource).not.toContain("key: 'pricing'")
    expect(homeViewSource).not.toContain("target: '#pricing'")
  })
})

describe('HomeView compliance — region notice replaces vision', () => {
  it('removes the vision section entirely', () => {
    expect(homeViewSource).not.toContain('vision')
    expect(homeViewSource).not.toContain('id="vision"')
  })

  it('keeps the region-notice section visually before feature content', () => {
    expect(homeViewSource).toContain('id="region-notice"')
    // Visual order is driven by Tailwind order-N classes, not DOM source position:
    // region (order-2) must sort before features (order-4).
    expect(homeViewSource).toContain('id="region-notice" class="relative order-2')
    expect(homeViewSource).toContain('id="features" class="relative order-4')
  })

  it('region CTA navigates to the legal notice page', () => {
    expect(homeViewSource).toContain('@click="goLegalNotice"')
    expect(homeViewSource).toMatch(/function goLegalNotice[\s\S]*?\/legal-notice/)
  })
})

describe('HomeView compliance — legal nav + footer', () => {
  it('adds a legal nav item targeting /legal-notice', () => {
    expect(homeViewSource).toContain("key: 'legal'")
    expect(homeViewSource).toContain("target: '/legal-notice'")
  })

  it('adds a footer legal line and CTA', () => {
    expect(homeViewSource).toContain('copy.footerLegalLine')
    expect(homeViewSource).toContain('copy.footerLegalCta')
  })
})

describe('LegalNoticeView', () => {
  it('shows company info without EIN or personal names', () => {
    expect(legalNoticeSource).toContain('Lumio Games LLC')
    expect(legalNoticeSource).toContain('State of Colorado, United States')
    expect(legalNoticeSource).toContain('admin@lumio.games')
    expect(legalNoticeSource).not.toMatch(/EIN/i)
    expect(legalNoticeSource).not.toContain('30-1494751')
  })
})
