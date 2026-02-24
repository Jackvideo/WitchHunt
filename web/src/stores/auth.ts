import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User } from '../types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)

  const isLoggedIn = computed(() => !!user.value?.token)

  function load() {
    const token = localStorage.getItem('token')
    const id = localStorage.getItem('user_id')
    const username = localStorage.getItem('username')
    if (token && id && username) {
      user.value = { id: Number(id), username, token }
    }
  }

  function setUser(u: User) {
    user.value = u
    localStorage.setItem('token', u.token)
    localStorage.setItem('user_id', String(u.id))
    localStorage.setItem('username', u.username)
  }

  function logout() {
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user_id')
    localStorage.removeItem('username')
  }

  async function apiFetch(url: string, options: RequestInit = {}) {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string> || {}),
    }
    if (user.value?.token) {
      headers['Authorization'] = `Bearer ${user.value.token}`
    }
    const res = await fetch(url, { ...options, headers })
    if (res.status === 401) {
      logout()
      throw new Error('session expired')
    }
    return res
  }

  load()

  return { user, isLoggedIn, setUser, logout, apiFetch }
})
