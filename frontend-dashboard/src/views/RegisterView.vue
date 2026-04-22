<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { http } from '@/api/http'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const router = useRouter()
const auth = useAuthStore()

const email = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function submit() {
  loading.value = true
  error.value = ''
  try {
    const res = await http.post('/auth/register', { email: email.value, password: password.value })
    const payload = res.data?.data ?? res.data
    auth.login(payload)
    router.replace('/dashboard')
  } catch (e: any) {
    error.value = e?.message || 'Register failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <form class="space-y-5" @submit.prevent="submit">
    <h1 class="brand-serif text-xl font-semibold text-ink-900">{{ t('auth.register.title') }}</h1>

    <label class="block">
      <span class="block text-xs text-ink-600 ui-sans mb-1">{{ t('auth.register.email') }}</span>
      <input
        v-model="email"
        type="email"
        required
        class="w-full rounded-md border border-ink-200 px-3 py-2 text-sm focus:outline-none focus:border-brand-500 focus:ring-2 focus:ring-brand-100 ui-sans"
      />
    </label>

    <label class="block">
      <span class="block text-xs text-ink-600 ui-sans mb-1">{{ t('auth.register.password') }}</span>
      <input
        v-model="password"
        type="password"
        required
        minlength="6"
        class="w-full rounded-md border border-ink-200 px-3 py-2 text-sm focus:outline-none focus:border-brand-500 focus:ring-2 focus:ring-brand-100 ui-sans"
      />
    </label>

    <p v-if="error" class="text-xs text-rose-500 ui-sans">{{ error }}</p>

    <button type="submit" class="btn-brand w-full" :disabled="loading">
      {{ loading ? t('common.loading') : t('auth.register.submit') }}
    </button>

    <RouterLink to="/login" class="block text-center text-xs text-brand-600 hover:underline ui-sans">
      {{ t('auth.register.toLogin') }}
    </RouterLink>
  </form>
</template>
