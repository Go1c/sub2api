import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { checkinAPI } from './api'
import type { CheckinResult, CheckinStatus } from './types'

export const useCheckinStore = defineStore('checkin', () => {
  const status = ref<CheckinStatus | null>(null)
  const lastResult = ref<CheckinResult | null>(null)
  const loading = ref(false)
  const submitting = ref(false)
  const initialized = ref(false)
  const enabled = computed(() => status.value?.enabled === true)
  const checkedInToday = computed(() => status.value?.checked_in_today === true)

  async function fetchStatus(): Promise<CheckinStatus> {
    loading.value = true
    try {
      const next = await checkinAPI.getUserStatus()
      status.value = next
      initialized.value = true
      return next
    } finally {
      loading.value = false
    }
  }

  async function checkIn(): Promise<CheckinResult> {
    submitting.value = true
    try {
      const result = await checkinAPI.checkIn()
      lastResult.value = result
      await fetchStatus()
      return result
    } finally {
      submitting.value = false
    }
  }

  function reset() {
    status.value = null
    lastResult.value = null
    loading.value = false
    submitting.value = false
    initialized.value = false
  }

  return { status, lastResult, loading, submitting, initialized, enabled, checkedInToday, fetchStatus, checkIn, reset }
})
