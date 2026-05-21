import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'

import LotteryDialog from '../LotteryDialog.vue'

const LotteryWheelStub = defineComponent({
  props: {
    disabled: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['start', 'spin-end'],
  setup(props, { emit, expose }) {
    expose({
      spinTo(index: number) {
        setTimeout(() => emit('spin-end', { index }), 0)
        return Promise.resolve()
      },
      isSpinning: () => false,
    })

    return () =>
      h(
        'button',
        {
          'data-test': 'spin-button',
          disabled: props.disabled,
          onClick: () => emit('start'),
        },
        'spin',
      )
  },
})

function mountDialog(drawFn: () => Promise<any>) {
  return mount(LotteryDialog, {
    props: {
      open: true,
      campaignTitle: '五月幸运转盘',
      subtitle: '登录就有机会，转一转赢取兑换码',
      prizeCount: 1,
      maxParticipants: 3,
      joined: 0,
      segments: [
        { label: '奖品 1', isPrize: true },
        { label: '谢谢参与', isPrize: false },
      ],
      drawFn,
    },
    global: {
      stubs: {
        LotteryWheel: LotteryWheelStub,
        Teleport: true,
      },
    },
  })
}

describe('LotteryDialog', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('shows a site-message claim tip after winning', async () => {
    const wrapper = mountDialog(() =>
      Promise.resolve({
        won: true,
        index: 0,
        label: '奖品 1',
        message: '恭喜你中奖！兑换码已通过站内信发放，请前往站内信领取。',
        site_message_id: 12,
        code: 'SECRET-CODE',
      } as any),
    )

    await wrapper.get('[data-test="spin-button"]').trigger('click')
    await flushPromises()
    vi.runAllTimers()
    await flushPromises()

    expect(wrapper.text()).toContain('站内信')
    expect(wrapper.find('a[href="/site-messages"]').exists()).toBe(true)
  })

  it('does not render the raw code in the dialog result', async () => {
    const wrapper = mountDialog(() =>
      Promise.resolve({
        won: true,
        index: 0,
        label: '奖品 1',
        message: '恭喜你中奖！兑换码已通过站内信发放，请前往站内信领取。',
        site_message_id: 12,
        code: 'SECRET-CODE',
      } as any),
    )

    await wrapper.get('[data-test="spin-button"]').trigger('click')
    await flushPromises()
    vi.runAllTimers()
    await flushPromises()

    expect(wrapper.text()).not.toContain('SECRET-CODE')
  })
})
