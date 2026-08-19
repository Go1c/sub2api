import { mount, RouterLinkStub } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it } from 'vitest'

import AppHeader from '../AppHeader.vue'

async function mountHeader() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: '/dashboard',
        component: { template: '<div />' },
        meta: {
          title: '仪表盘',
          description: '欢迎回来！这是您账户的概览。'
        }
      },
      {
        path: '/home',
        component: { template: '<div />' }
      },
      {
        path: '/models',
        component: { template: '<div />' }
      },
      {
        path: '/login',
        component: { template: '<div />' }
      }
    ]
  })
  router.push('/dashboard')
  await router.isReady()

  const i18n = createI18n({
    legacy: false,
    locale: 'zh-CN',
    messages: {
      'zh-CN': {
        nav: {
          docs: '文档'
        }
      }
    }
  })

  return mount(AppHeader, {
    global: {
      plugins: [createPinia(), router, i18n],
      stubs: {
        AnnouncementBell: true,
        Icon: true,
        LocaleSwitcher: true,
        RouterLink: RouterLinkStub,
        SubscriptionProgressMini: true
      }
    }
  })
}

describe('AppHeader home navigation', () => {
  it('links the former pricing slot directly to model market', async () => {
    const wrapper = await mountHeader()

    const links = wrapper.findAllComponents(RouterLinkStub)
    const modelMarketLink = links.find((routerLink) => routerLink.text() === '模型广场')

    expect(wrapper.text()).not.toContain('定价')
    expect(wrapper.text()).not.toContain('特性')
    expect(modelMarketLink).toBeTruthy()
    expect(modelMarketLink?.props('to')).toEqual({ path: '/models' })
    expect(
      links.some((routerLink) => JSON.stringify(routerLink.props('to')).includes('#pricing'))
    ).toBe(false)
    expect(
      links.some((routerLink) => JSON.stringify(routerLink.props('to')).includes('#features'))
    ).toBe(false)
  })

  it('renders Image2 generation as an authenticated handoff navigation link', async () => {
    const wrapper = await mountHeader()

    const link = wrapper
      .findAllComponents(RouterLinkStub)
      .find((routerLink) => routerLink.text() === 'Image2生图')

    expect(wrapper.text()).not.toContain('技术支持')
    expect(link).toBeTruthy()
    expect(link?.props('to')).toEqual({
      path: '/login',
      query: {
        handoff: '1',
        return_to: 'https://img.lumio.games/'
      }
    })
  })

  it('renders Codex download as an outbound link to bestcodex.app', async () => {
    const wrapper = await mountHeader()
    const links = wrapper.findAll('a').filter((anchor) => anchor.text().includes('Codex 下载'))

    expect(links.length).toBeGreaterThanOrEqual(1)
    for (const link of links) {
      expect(link.attributes('href')).toBe('https://bestcodex.app/')
      expect(link.attributes('target')).toBe('_blank')
      expect(link.attributes('rel')).toContain('noopener')
    }
  })
})
