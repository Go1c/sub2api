import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const featureDir = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const srcDir = resolve(featureDir, '../..')

function source(path: string): string {
  return readFileSync(resolve(srcDir, path), 'utf8')
}

describe('check-in integration surface', () => {
  it('registers dedicated user and admin routes', () => {
    const router = source('router/index.ts')
    expect(router).toContain("path: '/checkin'")
    expect(router).toContain("import('@/features/checkin/CheckinView.vue')")
    expect(router).toContain("path: '/admin/affiliates/checkins'")
    expect(router).toContain("import('@/features/checkin/AdminCheckinView.vue')")
  })

  it('loads user status after authentication and resets it on logout', () => {
    const app = source('App.vue')
    expect(app).toContain('useCheckinStore')
    expect(app).toContain('checkinStore.fetchStatus()')
    expect(app).toContain('checkinStore.reset()')
  })

  it('mounts the independent settings card without adding shared settings fields', () => {
    const settings = source('views/admin/SettingsView.vue')
    expect(settings).toContain('<CheckinSettingsCard')
    expect(settings).toContain("import CheckinSettingsCard from '@/features/checkin/CheckinSettingsCard.vue'")
  })

  it('keeps admin check-in history reachable while gating existing affiliate children', () => {
    const sidebar = source('components/layout/AppSidebar.vue')
    expect(sidebar).toContain('<SidebarCheckinCard :collapsed="sidebarCollapsed"')
    expect(sidebar).toContain("path: '/checkin', label: t('nav.checkin')")
    expect(sidebar).toContain("path: '/admin/affiliates/checkins', label: t('nav.checkinRecords')")
    expect(sidebar).toMatch(/path: '\/admin\/affiliates\/invites'[^}]+featureFlag: flagAffiliate/)
  })

  it('merges the feature-owned English and Chinese locale overlays', () => {
    const i18n = source('i18n/index.ts')
    expect(i18n).toContain("import('@/features/checkin/locales/en')")
    expect(i18n).toContain("import('@/features/checkin/locales/zh')")
  })
})
