<template>
  <div class="flex min-h-screen items-center justify-center bg-gray-50 px-4 dark:bg-dark-900">
    <p class="text-sm text-gray-600 dark:text-gray-300">登录中…</p>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { exchangeAuthBridge, parseAuthBridgeHash } from '@/utils/authBridge'

const router = useRouter()
const authStore = useAuthStore()

function stripTokenHash() {
  const nextUrl = `${window.location.pathname}${window.location.search}`
  window.history.replaceState(window.history.state, '', nextUrl)
}

onMounted(async () => {
  const { token, redirect } = parseAuthBridgeHash(window.location.hash)
  stripTokenHash()

  if (!token) {
    await router.replace({ path: '/login', query: { redirect } })
    return
  }

  try {
    const response = await exchangeAuthBridge(token)
    authStore.applyAuthResponse(response)
    await router.replace(redirect)
  } catch {
    await router.replace({ path: '/login', query: { redirect } })
  }
})
</script>
