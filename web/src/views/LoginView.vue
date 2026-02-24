<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()

const username = ref('')
const password = ref('')
const isRegister = ref(false)
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true

  try {
    const endpoint = isRegister.value ? '/api/auth/register' : '/api/auth/login'
    const res = await fetch(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: username.value, password: password.value }),
    })
    const data = await res.json()

    if (!res.ok) {
      error.value = data.error || 'request failed'
      return
    }

    if (isRegister.value) {
      isRegister.value = false
      error.value = ''
      return
    }

    auth.setUser({ id: data.id, username: data.username, token: data.token })
    router.push('/lobby')
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'network error'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex-1 flex items-center justify-center p-4">
    <div class="w-full max-w-sm space-y-6">
      <h1 class="text-3xl font-bold text-center text-primary-light">WitchHunt</h1>
      <p class="text-center text-sm text-gray-400">
        {{ isRegister ? '创建新账号' : '登录你的账号' }}
      </p>

      <form class="space-y-4" @submit.prevent="submit">
        <input
          v-model="username"
          type="text"
          placeholder="用户名"
          required
          minlength="2"
          maxlength="32"
          class="w-full px-4 py-3 bg-surface-light rounded-lg border border-gray-600 focus:border-primary-light focus:outline-none transition"
        />
        <input
          v-model="password"
          type="password"
          placeholder="密码"
          required
          minlength="6"
          maxlength="64"
          class="w-full px-4 py-3 bg-surface-light rounded-lg border border-gray-600 focus:border-primary-light focus:outline-none transition"
        />

        <p v-if="error" class="text-red-400 text-sm text-center">{{ error }}</p>

        <button
          type="submit"
          :disabled="loading"
          class="w-full py-3 bg-primary hover:bg-primary-light rounded-lg font-medium transition disabled:opacity-50"
        >
          {{ loading ? '...' : isRegister ? '注册' : '登录' }}
        </button>
      </form>

      <p class="text-center text-sm text-gray-400">
        {{ isRegister ? '已有账号？' : '没有账号？' }}
        <button class="text-primary-light hover:underline" @click="isRegister = !isRegister; error = ''">
          {{ isRegister ? '去登录' : '去注册' }}
        </button>
      </p>
    </div>
  </div>
</template>
