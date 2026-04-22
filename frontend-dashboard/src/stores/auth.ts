import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

const TOKEN_KEY = 'dc_token'
const USER_KEY = 'dc_user'

export interface User {
  id: string
  email: string
  name: string
  plan: string
  joinedAt: string
  balance: number
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem(TOKEN_KEY))
  const user = ref<User | null>(
    localStorage.getItem(USER_KEY) ? JSON.parse(localStorage.getItem(USER_KEY)!) : null
  )

  const isAuthed = computed(() => !!token.value)

  function login(payload: { token: string; user: User }) {
    token.value = payload.token
    user.value = payload.user
    localStorage.setItem(TOKEN_KEY, payload.token)
    localStorage.setItem(USER_KEY, JSON.stringify(payload.user))
  }

  function logout() {
    token.value = null
    user.value = null
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
  }

  return { token, user, isAuthed, login, logout }
})
