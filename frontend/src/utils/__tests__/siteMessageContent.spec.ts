import { describe, expect, it } from 'vitest'

import {
  publicHttpsImageUrl,
  safeLotteryPromoImageUrl,
  splitSiteMessageContent,
} from '../siteMessageContent'

describe('siteMessageContent', () => {
  it('accepts public https image URLs and rejects unsafe schemes', () => {
    expect(publicHttpsImageUrl('https://cdn.example.com/qr.png')).toBe('https://cdn.example.com/qr.png')
    expect(publicHttpsImageUrl('http://cdn.example.com/qr.png')).toBe('')
    expect(publicHttpsImageUrl('javascript:alert(1)')).toBe('')
    expect(publicHttpsImageUrl('data:image/png;base64,abc')).toBe('')
    expect(publicHttpsImageUrl('https://user:pass@cdn.example.com/qr.png')).toBe('')
  })

  it('allows same-origin relative lottery preview images', () => {
    expect(safeLotteryPromoImageUrl('/lottery-preview-qr.svg')).toBe('/lottery-preview-qr.svg')
    expect(safeLotteryPromoImageUrl('//cdn.example.com/qr.png')).toBe('')
  })

  it('turns a standalone https line into an image part', () => {
    const parts = splitSiteMessageContent(
      '你在「五月」中中奖。\n\n兑换码：LUCK-001\n\nhttps://cdn.example.com/qr.png',
    )

    expect(parts).toEqual([
      { type: 'text', text: '你在「五月」中中奖。\n\n兑换码：LUCK-001\n' },
      { type: 'image', url: 'https://cdn.example.com/qr.png' },
    ])
  })

  it('keeps http and inline urls as text', () => {
    expect(splitSiteMessageContent('see http://evil.example/x.png')).toEqual([
      { type: 'text', text: 'see http://evil.example/x.png' },
    ])
    expect(splitSiteMessageContent('封面 https://cdn.example.com/qr.png 在句子里')).toEqual([
      { type: 'text', text: '封面 https://cdn.example.com/qr.png 在句子里' },
    ])
  })
})
