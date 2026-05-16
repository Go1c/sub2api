import { describe, expect, it } from 'vitest'
import {
  PROVIDER_CONFIG_FIELDS,
  PROVIDER_SUPPORTED_TYPES,
} from '@/components/payment/providerConfig'

function findField(key: string) {
  const fields = PROVIDER_CONFIG_FIELDS.wxpay || []
  return fields.find(field => field.key === key)
}

describe('PROVIDER_CONFIG_FIELDS.wxpay', () => {
  it('keeps admin form validation aligned with backend-required credentials', () => {
    expect(findField('publicKeyId')?.optional).toBeFalsy()
    expect(findField('certSerial')?.optional).toBeFalsy()
  })

  it('only keeps the simplified visible credential set in the admin form', () => {
    expect(findField('mpAppId')).toBeUndefined()
    expect(findField('h5AppName')).toBeUndefined()
    expect(findField('h5AppUrl')).toBeUndefined()
  })
})

describe('Mapay provider admin config', () => {
  it('supports the existing user-facing Alipay and WeChat payment methods only', () => {
    expect(PROVIDER_SUPPORTED_TYPES.mapay).toEqual(['alipay', 'wxpay'])
  })

  it('uses the same required credential shape as Mapay/EasyPay-style gateways', () => {
    const fields = PROVIDER_CONFIG_FIELDS.mapay || []

    expect(fields.map(field => field.key)).toEqual([
      'pid',
      'pkey',
      'apiBase',
      'channelId',
      'channelIdAlipay',
      'channelIdWxpay',
    ])
    expect(fields.find(field => field.key === 'pid')?.optional).toBeFalsy()
    expect(fields.find(field => field.key === 'pkey')?.sensitive).toBe(true)
    expect(fields.find(field => field.key === 'apiBase')?.optional).toBeFalsy()
    expect(fields.find(field => field.key === 'channelId')?.optional).toBe(true)
    expect(fields.find(field => field.key === 'channelIdAlipay')?.optional).toBe(true)
    expect(fields.find(field => field.key === 'channelIdWxpay')?.optional).toBe(true)
  })
})
