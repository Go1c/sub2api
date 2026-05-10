import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

import { siteMessagesAPI } from '@/api/siteMessages'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

export const useSiteMessageStore = defineStore('siteMessages', () => {
  const unreadCount = ref(0)
  const loadingUnread = ref(false)

  const hasUnread = computed(() => unreadCount.value > 0)
  const featureEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.siteMessages))

  function reset(): void {
    unreadCount.value = 0
    loadingUnread.value = false
  }

  async function refreshUnreadCount(): Promise<number> {
    const authStore = useAuthStore()
    const appStore = useAppStore()

    if (!authStore.isAuthenticated || appStore.backendModeEnabled || !featureEnabled.value) {
      unreadCount.value = 0
      return 0
    }

    loadingUnread.value = true
    try {
      const response = await siteMessagesAPI.getUnreadCount()
      unreadCount.value = Math.max(0, Number(response.count) || 0)
      return unreadCount.value
    } catch (error: unknown) {
      const code = (error as { code?: string })?.code
      if (code === 'SITE_MESSAGES_DISABLED') {
        unreadCount.value = 0
      }
      throw error
    } finally {
      loadingUnread.value = false
    }
  }

  function setUnreadCount(count: number): void {
    unreadCount.value = Math.max(0, Number(count) || 0)
  }

  function decrementUnreadCount(): void {
    unreadCount.value = Math.max(0, unreadCount.value - 1)
  }

  return {
    unreadCount,
    loadingUnread,
    hasUnread,
    featureEnabled,
    refreshUnreadCount,
    setUnreadCount,
    decrementUnreadCount,
    reset,
  }
})
