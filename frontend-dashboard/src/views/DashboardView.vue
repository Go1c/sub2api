<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { http } from '@/api/http'
import { useAuthStore } from '@/stores/auth'
import UserDashboardStats from '@/components/dashboard/UserDashboardStats.vue'
import UserDashboardCharts from '@/components/dashboard/UserDashboardCharts.vue'
import UserDashboardRecentUsage from '@/components/dashboard/UserDashboardRecentUsage.vue'
import UserDashboardQuickActions from '@/components/dashboard/UserDashboardQuickActions.vue'
import RechargeModal from '@/components/dashboard/RechargeModal.vue'

const { t } = useI18n()
const auth = useAuthStore()

const stats = ref({ balance: 0, todayRequests: 0, todayTokens: 0, activeKeys: 0 })
const trend = ref<{ labels: string[]; requests: number[]; tokens: number[] }>({
  labels: [],
  requests: [],
  tokens: []
})
const recent = ref<Array<{ id: string; time: string; model: string; tokens: number; status: string }>>([])
const loading = ref(true)
const rechargeOpen = ref(false)

async function load() {
  loading.value = true
  const [s, tr, rc] = await Promise.all([
    http.get('/user/dashboard/stats'),
    http.get('/user/dashboard/trend'),
    http.get('/user/dashboard/recent')
  ])
  stats.value = s.data.data
  trend.value = tr.data.data
  recent.value = rc.data.data
  loading.value = false
}

onMounted(load)
</script>

<template>
  <div class="space-y-5 animate-fade-in">
    <header class="bg-navy-gradient rounded-lg text-white p-6 shadow-brand">
      <h1 class="brand-serif text-2xl font-semibold">
        {{ t('dashboard.welcome', { name: auth.user?.name ?? 'Explorer' }) }}
      </h1>
      <p class="mt-1 text-sm text-white/70 ui-sans">{{ t('dashboard.subtitle') }}</p>
    </header>

    <UserDashboardStats
      :balance="stats.balance"
      :today-requests="stats.todayRequests"
      :today-tokens="stats.todayTokens"
      :active-keys="stats.activeKeys"
    />

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-5">
      <div class="lg:col-span-2">
        <UserDashboardCharts
          :labels="trend.labels"
          :requests="trend.requests"
          :tokens="trend.tokens"
        />
      </div>
      <div>
        <UserDashboardQuickActions @recharge="rechargeOpen = true" />
      </div>
    </div>

    <UserDashboardRecentUsage :items="recent as any" />

    <RechargeModal :open="rechargeOpen" @close="rechargeOpen = false" />
  </div>
</template>
