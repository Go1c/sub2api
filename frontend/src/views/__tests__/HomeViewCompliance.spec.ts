import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const homeViewSource = readFileSync(resolve(here, '../HomeView.vue'), 'utf8')
const legalNoticeSource = readFileSync(resolve(here, '../public/LegalNoticeView.vue'), 'utf8')

function pricingSection(): string {
  const start = homeViewSource.indexOf('id="pricing"')
  const end = homeViewSource.indexOf('</section>', start)
  return homeViewSource.slice(start, end)
}

describe('HomeView compliance — pricing units', () => {
  it('renders LumioAPI price unit from a locale-aware copy field, not a hardcoded CNY', () => {
    const section = pricingSection()
    expect(section).toContain('{{ copy.pricingCurrency.unit }}')
    // No hardcoded CNY span left in the pricing table body
    expect(section).not.toContain('>CNY</span>')
  })

  it('formats prices without a ¥ currency symbol', () => {
    expect(homeViewSource).toMatch(/function formatCny[\s\S]*?return value\.toFixed\(2\)/)
    // formatConverted keeps the ×7 logic but drops the ¥ symbol
    expect(homeViewSource).toMatch(/function formatConverted[\s\S]*?value \* 7/)
    expect(homeViewSource).not.toContain('`¥${value.toFixed(2)}`')
    expect(homeViewSource).not.toContain('`¥${converted')
  })

  it('uses 积分 / Credits wording in pricing copy instead of 人民币 / CNY', () => {
    expect(homeViewSource).toContain('积分（Credits）')
    expect(homeViewSource).toContain('cny: \'积分 Credits\'')
    expect(homeViewSource).toContain('1 积分 (Credit) = $1 美元额度')
  })
})

describe('HomeView compliance — region notice replaces vision', () => {
  it('removes the vision section entirely', () => {
    expect(homeViewSource).not.toContain('vision')
    expect(homeViewSource).not.toContain('id="vision"')
  })

  it('adds a region-notice section visually ordered before pricing via order classes', () => {
    expect(homeViewSource).toContain('id="region-notice"')
    // Visual order is driven by Tailwind order-N classes, not DOM source position:
    // region (order-2) must sort before pricing (order-3).
    expect(homeViewSource).toContain('id="region-notice" class="relative order-2')
    expect(homeViewSource).toContain('id="pricing" class="relative order-3')
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
