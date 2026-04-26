import { mount } from '@vue/test-utils'
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
        RouterLink: true,
        SubscriptionProgressMini: true
      }
    }
  })
}

describe('AppHeader home navigation', () => {
  it('renders Image2 generation as an external navigation link', async () => {
    const wrapper = await mountHeader()

    const link = wrapper.findAll('a').find((anchor) => anchor.text() === 'Image2生图')

    expect(wrapper.text()).not.toContain('技术支持')
    expect(link).toBeTruthy()
    expect(link?.attributes('href')).toBe('https://img.lumio.games/')
    expect(link?.attributes('target')).toBe('_blank')
    expect(link?.attributes('rel')).toBe('noopener noreferrer')
  })
})
