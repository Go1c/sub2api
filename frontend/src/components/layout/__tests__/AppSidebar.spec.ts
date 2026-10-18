import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
  })
})

describe('AppSidebar scroll position persistence', () => {
  it('binds a template ref to the sidebar nav element', () => {
    expect(componentSource).toContain('ref="sidebarNavRef"')
    expect(componentSource).toContain('sidebar-nav')
  })

  it('declares sidebarNavRef in script setup', () => {
    expect(componentSource).toContain("const sidebarNavRef = ref<HTMLElement | null>(null)")
  })

  it('saves scroll position on beforeUnmount', () => {
    expect(componentSource).toContain('onBeforeUnmount')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('sidebarNavRef.value.scrollTop')
  })

  it('restores scroll position on mount', () => {
    expect(componentSource).toContain('onMounted')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('nextTick')
  })
})

describe('AppSidebar collapsible groups', () => {
  it('lets the user collapse a group even while a child route is active', () => {
    // The expand state must come from the user's override first, falling back
    // to the active-route heuristic only when the user has not clicked yet.
    expect(componentSource).toContain('const groupExpandOverrides = ref<Map<string, boolean>>(new Map())')
    expect(componentSource).not.toContain('expandedGroups.value.has(item.path) || isGroupActive(item)')
  })
})

describe('AppSidebar header styles', () => {
  it('does not clip the version badge dropdown', () => {
    const sidebarHeaderBlockMatch = styleSource.match(/\.sidebar-header\s*\{[\s\S]*?\n {2}\}/)
    const sidebarBrandBlockMatch = componentSource.match(/\.sidebar-brand\s*\{[\s\S]*?\n\}/)

    expect(sidebarHeaderBlockMatch).not.toBeNull()
    expect(sidebarBrandBlockMatch).not.toBeNull()
    expect(sidebarHeaderBlockMatch?.[0]).not.toContain('@apply overflow-hidden;')
    expect(sidebarBrandBlockMatch?.[0]).not.toContain('overflow: hidden;')
  })
})

describe('AppSidebar unread indicator styles', () => {
  it('uses a larger animated dot for unread site messages', () => {
    const unreadDotBlockMatch = componentSource.match(/\.sidebar-unread-dot\s*\{[\s\S]*?\n\}/)

    expect(unreadDotBlockMatch).not.toBeNull()
    expect(unreadDotBlockMatch?.[0]).toContain('height: 0.625rem;')
    expect(unreadDotBlockMatch?.[0]).toContain('width: 0.625rem;')
    expect(unreadDotBlockMatch?.[0]).toContain('animation: sidebarUnreadPulse')
    expect(componentSource).toContain('@keyframes sidebarUnreadPulse')
  })
})

describe('AppSidebar invoice navigation', () => {
  it('gates the user invoice tab by per-user invoice access and exposes admin invoice records', () => {
    expect(componentSource).toContain('flagInvoiceAccess')
    expect(componentSource).toContain("path: '/invoices'")
    expect(componentSource).toContain("path: '/admin/invoices'")
    expect(componentSource).toContain("t('nav.invoices')")
    expect(componentSource).toContain("t('nav.invoiceManagement')")
  })
})

describe('AppSidebar admin subscription navigation', () => {
  it('shows payment plans under subscriptions but still gates them by payment settings', () => {
    const paymentPlansItemMatch = componentSource.match(
      /\{ path: '\/admin\/subscriptions\/plans'[^}]*\}/,
    )

    expect(paymentPlansItemMatch).not.toBeNull()
    expect(paymentPlansItemMatch?.[0]).toContain("t('nav.paymentPlans')")
    expect(paymentPlansItemMatch?.[0]).toContain('featureFlag: flagAdminPayment')
    expect(componentSource).not.toContain("path: '/admin/orders/plans', label: t('nav.paymentPlans')")
  })
})
