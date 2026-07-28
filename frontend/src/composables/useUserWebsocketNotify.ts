import { ref, watch, type Ref } from 'vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useSiteMessageStore } from '@/stores/siteMessages'
import { useAnnouncementStore } from '@/stores/announcements'
import { useRouter } from 'vue-router'

export type UserWSEventType =
  | 'connected'
  | 'test'
  | 'balance_low'
  | 'site_message'
  | 'announcement'
  | string

export interface UserWSEvent {
  type: UserWSEventType
  title?: string
  message?: string
  data?: Record<string, unknown>
  timestamp?: number
}

const USER_WS_PROTOCOL = 'sub2api-user'
const WS_CLOSE_DISABLED = 4001

let sharedWs: WebSocket | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let shouldRun = false
let started = false

function buildWsUrl(): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/api/v1/user/ws/notifications`
}

function clearReconnect() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
}

function scheduleReconnect(connect: () => void) {
  clearReconnect()
  reconnectTimer = setTimeout(connect, 3000)
}

/**
 * Full-site user WebSocket connection. Call once from App.vue.
 * Connects when authenticated non-admin user has websocket_notify_enabled.
 */
export function useUserWebsocketNotify() {
  const authStore = useAuthStore()
  const appStore = useAppStore()
  const siteMessagesStore = useSiteMessageStore()
  const announcementStore = useAnnouncementStore()
  const router = useRouter()
  const status: Ref<'off' | 'connecting' | 'connected' | 'error'> = ref('off')

  const handleEvent = (event: UserWSEvent) => {
    const title = event.title || '通知'
    const message = event.message || ''
    switch (event.type) {
      case 'connected':
        // silent
        break
      case 'test':
        appStore.showSuccess(message || title)
        break
      case 'balance_low':
        appStore.showError(`${title}: ${message}`)
        break
      case 'site_message':
        appStore.showInfo(message ? `${title}: ${message}` : title)
        void siteMessagesStore.refreshUnreadCount()
        break
      case 'announcement':
        appStore.showInfo(message ? `${title}: ${message}` : title)
        void announcementStore.fetchAnnouncements(true)
        break
      default:
        if (message || title) {
          appStore.showInfo(message ? `${title}: ${message}` : title)
        }
    }
  }

  const disconnect = () => {
    shouldRun = false
    clearReconnect()
    if (sharedWs) {
      try {
        sharedWs.close()
      } catch {
        /* ignore */
      }
      sharedWs = null
    }
    status.value = 'off'
  }

  const connect = () => {
    const user = authStore.user
    if (!authStore.isAuthenticated || !user || user.role === 'admin') {
      disconnect()
      return
    }
    if (!user.websocket_notify_enabled) {
      disconnect()
      return
    }
    const token = authStore.token
    if (!token) {
      disconnect()
      return
    }

    shouldRun = true
    if (
      sharedWs &&
      (sharedWs.readyState === WebSocket.OPEN || sharedWs.readyState === WebSocket.CONNECTING)
    ) {
      return
    }

    status.value = 'connecting'
    const protocols = [USER_WS_PROTOCOL, `jwt.${token}`]
    const ws = new WebSocket(buildWsUrl(), protocols)
    sharedWs = ws

    ws.onopen = () => {
      if (sharedWs !== ws) return
      status.value = 'connected'
    }

    ws.onmessage = (ev) => {
      try {
        const data = JSON.parse(String(ev.data)) as UserWSEvent
        handleEvent(data)
      } catch {
        /* ignore malformed */
      }
    }

    ws.onerror = () => {
      if (sharedWs === ws) status.value = 'error'
    }

    ws.onclose = (ev) => {
      if (sharedWs === ws) {
        sharedWs = null
        status.value = 'off'
      }
      if (ev.code === WS_CLOSE_DISABLED) {
        shouldRun = false
        return
      }
      if (shouldRun) {
        scheduleReconnect(connect)
      }
    }
  }

  const ensureStarted = () => {
    if (started) return
    started = true
    watch(
      () =>
        [
          authStore.isAuthenticated,
          authStore.user?.id,
          authStore.user?.role,
          authStore.user?.websocket_notify_enabled,
          authStore.token
        ] as const,
      () => {
        connect()
      },
      { immediate: true }
    )
  }

  return {
    status,
    connect,
    disconnect,
    ensureStarted,
    // exposed for tests / deep links
    openSiteMessages: () => router.push('/site-messages')
  }
}
