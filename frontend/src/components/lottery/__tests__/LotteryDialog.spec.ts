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
    size: {
      type: Number,
      default: 320,
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
          'data-size': props.size,
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
    vi.unstubAllGlobals()
    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      value: 1024,
    })
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
    expect(wrapper.find('[data-test="lottery-result-modal"]').exists()).toBe(true)
    const claimLink = wrapper.get('a[href="/site-messages"]')
    expect(claimLink.attributes('class')).toContain('from-[#4f8cff]')
    expect(claimLink.attributes('class')).not.toContain('from-amber')
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

  it('uses a smaller wheel on narrow mobile screens', () => {
    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      value: 320,
    })

    const wrapper = mountDialog(() =>
      Promise.resolve({
        won: false,
        index: 1,
        label: '谢谢参与',
        message: '很遗憾，这次没有中奖。',
      }),
    )

    expect(Number(wrapper.get('[data-test="spin-button"]').attributes('data-size'))).toBeLessThan(280)
  })

  it('shows a skippable WeChat promo step before the wheel', async () => {
    const wrapper = mount(LotteryDialog, {
      props: {
        open: true,
        campaignTitle: '五月幸运转盘',
        subtitle: '登录就有机会，转一转赢取兑换码',
        prizeCount: 1,
        maxParticipants: 3,
        joined: 0,
        promoText: '关注公众号领福利',
        promoImageUrl: '/lottery-preview-qr.svg',
        segments: [
          { label: '奖品', isPrize: true },
          { label: '谢谢参与', isPrize: false },
        ],
        drawFn: () => Promise.resolve({
          won: true,
          index: 0,
          label: '奖品',
          message: '恭喜你中奖！兑换码已通过站内信发放，请前往站内信领取。',
        }),
      },
      global: {
        stubs: {
          LotteryWheel: LotteryWheelStub,
          Teleport: true,
        },
      },
    })

    expect(wrapper.get('[data-test="lottery-promo-step"]').text()).toContain('去抽奖')
    expect(wrapper.find('[data-test="spin-button"]').exists()).toBe(false)

    await wrapper.get('[data-test="lottery-promo-continue"]').trigger('click')
    expect(wrapper.find('[data-test="lottery-promo-step"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="spin-button"]').exists()).toBe(true)
  })

  it('shows the promo poster on winning results only', async () => {
    const wrapper = mount(LotteryDialog, {
      props: {
        open: true,
        campaignTitle: '五月幸运转盘',
        subtitle: '登录就有机会，转一转赢取兑换码',
        prizeCount: 1,
        maxParticipants: 3,
        joined: 0,
        promoText: '关注公众号领福利',
        promoImageUrl: 'https://cdn.example.com/qr.png',
        segments: [
          { label: '奖品', isPrize: true },
          { label: '谢谢参与', isPrize: false },
        ],
        drawFn: () => Promise.resolve({
          won: true,
          index: 0,
          label: '奖品',
          message: '恭喜你中奖！兑换码已通过站内信发放，请前往站内信领取。',
        }),
      },
      global: {
        stubs: {
          LotteryWheel: LotteryWheelStub,
          Teleport: true,
        },
      },
    })

    await wrapper.get('[data-test="lottery-promo-continue"]').trigger('click')
    await wrapper.get('[data-test="spin-button"]').trigger('click')
    await flushPromises()
    vi.runAllTimers()
    await flushPromises()

    const promo = wrapper.get('[data-test="lottery-result-promo"]')
    expect(promo.get('img').attributes('src')).toBe('https://cdn.example.com/qr.png')
    expect(promo.text()).toContain('关注公众号领福利')
  })

  it('does not show the promo poster after a losing draw', async () => {
    const wrapper = mount(LotteryDialog, {
      props: {
        open: true,
        campaignTitle: '五月幸运转盘',
        subtitle: '登录就有机会，转一转赢取兑换码',
        prizeCount: 1,
        maxParticipants: 3,
        joined: 0,
        promoText: '关注公众号领福利',
        promoImageUrl: 'https://cdn.example.com/qr.png',
        segments: [
          { label: '奖品', isPrize: true },
          { label: '谢谢参与', isPrize: false },
        ],
        drawFn: () => Promise.resolve({
          won: false,
          index: 1,
          label: '谢谢参与',
          message: '很遗憾，这次没有中奖。',
        }),
      },
      global: {
        stubs: {
          LotteryWheel: LotteryWheelStub,
          Teleport: true,
        },
      },
    })

    await wrapper.get('[data-test="lottery-promo-continue"]').trigger('click')
    await wrapper.get('[data-test="spin-button"]').trigger('click')
    await flushPromises()
    vi.runAllTimers()
    await flushPromises()

    expect(wrapper.find('[data-test="lottery-result-promo"]').exists()).toBe(false)
  })

  it('shows the promo poster on winning results only', async () => {
    const wrapper = mount(LotteryDialog, {
      props: {
        open: true,
        campaignTitle: '五月幸运转盘',
        subtitle: '登录就有机会，转一转赢取兑换码',
        prizeCount: 1,
        maxParticipants: 3,
        joined: 0,
        promoText: '关注公众号领福利',
        promoImageUrl: 'https://cdn.example.com/qr.png',
        segments: [
          { label: '奖品', isPrize: true },
          { label: '谢谢参与', isPrize: false },
        ],
        drawFn: () => Promise.resolve({
          won: true,
          index: 0,
          label: '奖品',
          message: '恭喜你中奖！兑换码已通过站内信发放，请前往站内信领取。',
        }),
      },
      global: {
        stubs: {
          LotteryWheel: LotteryWheelStub,
          Teleport: true,
        },
      },
    })

    await wrapper.get('[data-test="lottery-promo-continue"]').trigger('click')
    await wrapper.get('[data-test="spin-button"]').trigger('click')
    await flushPromises()
    vi.runAllTimers()
    await flushPromises()

    const promo = wrapper.get('[data-test="lottery-result-promo"]')
    expect(promo.get('img').attributes('src')).toBe('https://cdn.example.com/qr.png')
    expect(promo.text()).toContain('关注公众号领福利')
  })

  it('does not show the promo poster after a losing draw', async () => {
    const wrapper = mount(LotteryDialog, {
      props: {
        open: true,
        campaignTitle: '五月幸运转盘',
        subtitle: '登录就有机会，转一转赢取兑换码',
        prizeCount: 1,
        maxParticipants: 3,
        joined: 0,
        promoText: '关注公众号领福利',
        promoImageUrl: 'https://cdn.example.com/qr.png',
        segments: [
          { label: '奖品', isPrize: true },
          { label: '谢谢参与', isPrize: false },
        ],
        drawFn: () => Promise.resolve({
          won: false,
          index: 1,
          label: '谢谢参与',
          message: '很遗憾，这次没有中奖。',
        }),
      },
      global: {
        stubs: {
          LotteryWheel: LotteryWheelStub,
          Teleport: true,
        },
      },
    })

    await wrapper.get('[data-test="lottery-promo-continue"]').trigger('click')
    await wrapper.get('[data-test="spin-button"]').trigger('click')
    await flushPromises()
    vi.runAllTimers()
    await flushPromises()

    expect(wrapper.find('[data-test="lottery-result-promo"]').exists()).toBe(false)
  })

  it('shows losing results in a dedicated result modal', async () => {
    const wrapper = mountDialog(() =>
      Promise.resolve({
        won: false,
        index: 1,
        label: '谢谢参与',
        message: '很遗憾，这次没有中奖。',
      }),
    )

    await wrapper.get('[data-test="spin-button"]').trigger('click')
    await flushPromises()
    vi.runAllTimers()
    await flushPromises()

    const modal = wrapper.get('[data-test="lottery-result-modal"]')
    expect(modal.text()).toContain('很遗憾')
    expect(modal.text()).toContain('下次再来')
  })
})
