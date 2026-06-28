import type { UserSubscription } from '@/types'

type Translate = (key: string, params?: Record<string, unknown>) => string

export function subscriptionDisplayName(subscription: UserSubscription, t: Translate): string {
  const planName = subscription.plan_name?.trim()
  if (planName) return planName

  const productName = subscription.plan_product_name?.trim()
  if (productName) return productName

  if (subscription.group?.name) return subscription.group.name

  if (subscription.group_id != null) {
    return t('payment.groupFallback', { id: subscription.group_id })
  }

  return t('userSubscriptions.creditPoolSubscription')
}
