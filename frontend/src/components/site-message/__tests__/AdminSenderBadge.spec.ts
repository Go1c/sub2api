import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import AdminSenderBadge from '@/components/site-message/AdminSenderBadge.vue'

describe('AdminSenderBadge', () => {
  it('renders a unique official marker for admin senders', () => {
    const wrapper = mount(AdminSenderBadge, {
      props: {
        isAdmin: true,
        label: '官方',
      },
    })

    expect(wrapper.get('[data-test="admin-sender-badge"]').text()).toContain('官方')
  })

  it('renders nothing for non-admin senders', () => {
    const wrapper = mount(AdminSenderBadge, {
      props: {
        isAdmin: false,
        label: '官方',
      },
    })

    expect(wrapper.find('[data-test="admin-sender-badge"]').exists()).toBe(false)
  })
})
